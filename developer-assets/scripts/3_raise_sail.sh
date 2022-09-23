#!/bin/bash

rm -r /tmp_mnt/anchors

[[ ! -d /chemotion/.git ]] && git clone $CHEMOTION_GIT chemotion
cd chemotion
git stash
git fetch --all
git -c advice.detachedHead=false checkout $BRANCH_OR_HASH

echo "Copying gems..."
cp -r /cargo/gems $GEM_HOME
echo "Copying yarn cache..."
cp -r /cargo/yarn $YARN_CACHE
echo "Copying node modules..."
cp -r /cargo/node_modules $NODE_MODULES

# make node modules
yarn install --modules-folder $NODE_MODULES --cache-folder $YARN_CACHE

# install ruby
gem install solargraph
bundle install --jobs=$(getconf _NPROCESSORS_ONLN)
if [[ $(grep -L passenger Gemfile) ]]; then bundle add passenger; fi

# prepare resources
cp public/welcome-message-sample.md public/welcome-message.md

if [ ! -f config/database.yml ]; then
    yq -o json "(.*.database=\"$PGDATABASE\") | (.*.host=\"$PGHOST\") | (.*.username=\"$PGUSER\") | (.*.password=\"$PGPASSWORD\")" config/database.yml.example | \
    yq -o json "(.test.database=\"chemotion-test\")" | yq -P -o yaml > /tmp/database.yml
    mv /tmp/database.yml config/database.yml
fi

if [ ! -f config/storage.yml ]; then
    cp config/storage.yml.example config/storage.yml
fi

if [ ! -f config/datacollectors.yml ]; then
    cp config/datacollectors.yml.example config/datacollectors.yml
fi

if [ ! -f .env ]; then
    cp .env.development .env
fi

while ! pg_isready; do echo "Waiting for database on [${PGHOST}]..."; sleep 3; done

echo "Creating DB..."
bundle exec rake db:create
echo "Migrating DB..."
bundle exec rake db:migrate

psql -c "CREATE EXTENSION IF NOT EXISTS \"pg_trgm\";"
psql -c "CREATE EXTENSION IF NOT EXISTS \"hstore\";"
psql -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"

seedCheck=$(psql -tA -c "select 'non-empty' from molecule_names limit 1;")
[[ "${seedCheck}" =~ "non-empty" ]] || ( bundle exec rake db:seed &>/tmp/seed.log; bundle exec rake ketcherails:import:common_templates )
bundle exec rake assets:precompile
bundle exec rake webpacker:compile

echo -e "Rails.application.configure do 
  if File.file?('/.dockerenv') == true
    host_ip = \`/sbin/ip route|awk '/default/ { print \$3 }'\`.strip
    config.web_console.whitelisted_ips << host_ip
  end
end" >> /chemotion/config/environments/development.rb

echo -e "#!/bin/bash
rm -f /tmp/pids/server.pid
bundle exec rails server -p 3000 -b 0.0.0.0" >> /chemotion/run_devcon.sh

chown -R $UID:$GID /chemotion