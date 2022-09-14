package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/chigopher/pathlib"
	"github.com/docker/docker/api/types"
	dc "github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var Reset = "\033[0m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Red = "\033[31m"

func init() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Green = ""
		Yellow = ""
		Red = ""
	}
}

func setDefaultValue(someValue string, defaultValue string) string {

	if string(someValue) == "\n" || string(someValue) == "" {
		someValue = string(defaultValue)
	}
	return someValue
}

// This function prompt user to type in different input parameters such as password, first name, last name, abbeviation.
func userDataInput() ([]byte, string, string, string) {
	var fname, lname, abbreviation string

	fmt.Println(Green + "Please type in the Password or press Enter for defaults" + Reset)
	passwd, err := term.ReadPassword(0)

	if err != nil {
		fmt.Println("An error occured while reading input. Please try again", err)
	}

	if string(passwd) == "" {
		fmt.Println("setting default password [chemotion]")
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println(Green + "Enter First name: " + Reset)
	fname, _ = reader.ReadString('\n')

	fmt.Println(Green + "Enter Last name: " + Reset)
	lname, _ = reader.ReadString('\n')

	fmt.Println(Green + "Enter a unique abbreviation." + Reset)
	abbreviation, _ = reader.ReadString('\n')

	fmt.Println(Green + "Processing..." + Reset)

	fname = strings.TrimSuffix(fname, "\n")
	lname = strings.TrimSuffix(lname, "\n")
	abbreviation = strings.TrimSuffix(abbreviation, "\n")

	return passwd, fname, lname, abbreviation
}

// setupScript function downloads a file from a specified url, tar it and move it to script directory inside a container.
//
// This function can be avoided, if scripts are direclty supplied with docker image of Chemotion.
func setupScript(url string, script_name string) {
	script := downloadFile(url, script_name)
	script_tar := getNewUniqueID() + ".tar"
	cmd := exec.Command("tar", "-cf", script_tar, script.String())
	cmd.Stdout = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("path: %s", cmd)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to call cmd.Run(): %v", err)
	}
	script.Remove()

	copyFilesInContainer(
		currentInstance,
		"eln",      //target service
		script_tar, //source
		"/script")  // destination

	pathlib.NewPath(script_tar).Remove()
}

func setUpDockerCleint() (context.Context, *dc.Client) {
	ctx := context.Background()
	cli, err := dc.NewClientWithOpts(dc.FromEnv, dc.WithAPIVersionNegotiation())
	panicCheck(err)
	return ctx, cli
}

func handleCreateUserLogic() {
	containerID := getContainerID_api(currentInstance, "eln")
	pathToScript := "/script/createScript.sh"

	url := "https://raw.githubusercontent.com/mehmood86/shellscripts/main/createScript.sh"
	script_name := "createScript.sh"
	setupScript(url, script_name)

	ctx, cli := setUpDockerCleint()

	var email string

	fmt.Println(Green + "Enter an Unique E-mail address for new user." + Reset)
	fmt.Scanln(&email)
	email = setDefaultValue(email, "eln-user@kit.edu")

	passwd, fname, lname, abbreviation := userDataInput()

	// Set default values if left empty
	fname = setDefaultValue(fname, "ELN")
	lname = setDefaultValue(lname, "User")
	abbreviation = setDefaultValue(abbreviation, "CU1")

	arg1 := email
	arg2 := passwd
	arg3 := fname
	arg4 := lname
	arg5 := abbreviation

	cmdStatementExecuteScript := []string{"bash", pathToScript, arg1, string(arg2), arg3, arg4, arg5}
	optionsCreateExecuteScript := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmdStatementExecuteScript,
	}

	rst_ExecuteScript, err := cli.ContainerExecCreate(ctx, containerID, optionsCreateExecuteScript)
	panicCheck(err)

	response_ExecuteScript, err := cli.ContainerExecAttach(ctx, rst_ExecuteScript.ID, types.ExecStartCheck{})
	panicCheck(err)

	defer response_ExecuteScript.Close()
	data, _ := io.ReadAll(response_ExecuteScript.Reader)

	if strings.Contains(string(data), "false") {
		fmt.Println(Red + "WARNING!" + Reset)
		fmt.Printf("Enter a unique (Non-Empty) E-Mail address and abbreviation.\n")
		fmt.Printf("Check if E-Mail: %s, or abbreviation: %s already exists\n", arg1, arg5)
	} else {
		fmt.Printf(Green+"User with E-Mail: %s created successfully.\n", email+Reset)
	}
}

func handleUpdateUserLogic() {
	containerID := getContainerID_api(currentInstance, "eln")
	pathToScript := "/script/updateScript.sh"

	// Download a file and copy it inside running container
	url := "https://raw.githubusercontent.com/mehmood86/shellscripts/main/updateScript.sh"
	script_name := "updateScript.sh"
	setupScript(url, script_name)

	ctx, cli := setUpDockerCleint()

	var email string
	fmt.Println(Yellow + "Enter E-Mail address of the user that need to be be updated." + Reset)
	fmt.Scanln(&email)

	if email == "" {
		log.Fatal(Red + "Aborting. Please provide an E-Mail to continue with update process." + Reset)
	} else {
		passwd, fname, lname, abbreviation := userDataInput()
		arg1 := email
		arg2 := passwd
		arg3 := fname
		arg4 := lname
		arg5 := abbreviation

		cmdStatementExecuteScript := []string{"bash", pathToScript, arg1, string(arg2), arg3, arg4, arg5}
		optionsCreateExecuteScript := types.ExecConfig{
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          cmdStatementExecuteScript,
		}

		rst_ExecuteScript, err := cli.ContainerExecCreate(ctx, containerID, optionsCreateExecuteScript)
		panicCheck(err)

		response_ExecuteScript, err := cli.ContainerExecAttach(ctx, rst_ExecuteScript.ID, types.ExecStartCheck{})
		panicCheck(err)

		defer response_ExecuteScript.Close()
		data, _ := io.ReadAll(response_ExecuteScript.Reader)
		fmt.Println(string(data))

	}
}

func handleDeleteUserLogic() {
	containerID := getContainerID_api(currentInstance, "eln")
	pathToScript := "/script/deleteScript.sh"

	// Download a file and copy it inside running container
	url := "https://raw.githubusercontent.com/mehmood86/shellscripts/main/deleteScript.sh"
	script_name := "deleteScript.sh"
	setupScript(url, script_name)

	ctx, cli := setUpDockerCleint()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println(Red + "WARNING: " + Yellow + "The user will be deleted." + Reset)
	fmt.Println("E-Mail address: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSuffix(email, "\n")
	if email == "" {
		log.Fatal(Red + "Aborting. Please provide an E-Mail to continue with delete process." + Reset)
	} else {
		cmdStatementExecuteScript := []string{"bash", pathToScript, email}
		optionsCreateExecuteScript := types.ExecConfig{
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          cmdStatementExecuteScript,
		}

		rst_ExecuteScript, err := cli.ContainerExecCreate(ctx, containerID, optionsCreateExecuteScript)
		panicCheck(err)

		response_ExecuteScript, err := cli.ContainerExecAttach(ctx, rst_ExecuteScript.ID, types.ExecStartCheck{})
		panicCheck(err)

		defer response_ExecuteScript.Close()
		data, _ := io.ReadAll(response_ExecuteScript.Reader)
		fmt.Println(string(data))
	}
}

// create a new user of type Admin, Person and Device (for now only type:Person is supported)
var createUserManagementInstanceRootCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c", "create"},
	Args:    cobra.NoArgs,
	Short:   "Manage user actions such as create, add, update and remove user and reset password for " + nameCLI,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle Add user logic here
		fmt.Println("Please create a user")
		// Some decorative text as guideline about what kind of parameters should be unique
		if ownCall(cmd) {
			handleCreateUserLogic()
		} else {
			handleCreateUserLogic()
		}
	},
}

// Update an existing user (i,e. first name, last name, password, abbrevitation)
var updateUserManagementInstanceRootCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"u", "update"},
	Args:    cobra.NoArgs,
	Short:   "Manage user such as create, add, update and remove user and reset password for " + nameCLI,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle Add update logic here
		if ownCall(cmd) {
			handleUpdateUserLogic()
		} else {
			handleUpdateUserLogic()
		}
	},
}

// Destroy a particular user from the user management list
var deleteUserManagementInstanceRootCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"d", "delete"},
	Args:    cobra.NoArgs,
	Short:   "Manage user such as create, add, update and remove user and reset password for " + nameCLI,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle delete user logic here
		if ownCall(cmd) {
			handleDeleteUserLogic()
		} else {
			handleDeleteUserLogic()
		}
	},
}

var listUserManagementInstanceRootCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l", "list"},
	Args:    cobra.NoArgs,
	Short:   "Manage user such as create, add, update and remove user and reset password for " + nameCLI,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle list all users logic here
		fmt.Println("List of all users are as follows")
		if ownCall(cmd) {
			fmt.Println("triggered by own call")
		} else {
			fmt.Println("triggered via cli menu")
		}
	},
}

func init() {
	usermanagementCmd.AddCommand(createUserManagementInstanceRootCmd)
	usermanagementCmd.AddCommand(deleteUserManagementInstanceRootCmd)
	usermanagementCmd.AddCommand(listUserManagementInstanceRootCmd)
	usermanagementCmd.AddCommand(updateUserManagementInstanceRootCmd)
}
