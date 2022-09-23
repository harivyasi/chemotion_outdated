#!/bin/sh

mkdir -p /tmp_mnt/anchors

mkdir tmp_chemotion
cd tmp_chemotion
git init -q
git remote add origin $CHEMOTION_GIT 
git fetch -q --depth 1 origin $BRANCH_OR_HASH
git checkout -q FETCH_HEAD

cp package.json              /tmp_mnt/anchors/.
cp yarn.lock                 /tmp_mnt/anchors/.
cp Gemfile                   /tmp_mnt/anchors/.
cp Gemfile.lock              /tmp_mnt/anchors/.

chown -R $UID:$GID /tmp_mnt