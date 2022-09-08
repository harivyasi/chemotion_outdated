package cli

import (
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

	if string(someValue) == "" {
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

	fmt.Println(Green + "First name: " + Reset)
	fmt.Scanln(&fname)
	fname = setDefaultValue(fname, "ELN")
	fmt.Println(Green + "Last name: " + Reset)
	fmt.Scanln(&lname)
	lname = setDefaultValue(lname, "User")
	fmt.Println(Yellow + "Please enter a unique abbreviation." + Reset)
	fmt.Println(Green + "abbriviation: " + Reset)
	fmt.Scanln(&abbreviation)
	abbreviation = setDefaultValue(abbreviation, "CU1")
	/*
		//optionaly print to the console, but not necessarily required
		fmt.Println("--------------------------------------------------")
		fmt.Println(Yellow + "Following data will be send to the database:" + Reset)
		fmt.Println(Yellow+"first name ", fname)
		fmt.Println(Yellow+"last name: ", lname)
		fmt.Println(Yellow+"abbreviation: ", abbreviation)
		fmt.Println(Reset + "--------------------------------------------------")
	*/
	return passwd, fname, lname, abbreviation
}

// Download a file from a specified url, tar it and move it to script directory inside a container
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

func handleCreateUserLogic() {
	containerID := getContainerID_api(currentInstance, "eln")
	pathToScript := "/script/createScript.sh"

	url := "https://raw.githubusercontent.com/mehmood86/shellscripts/main/createScript.sh"
	script_name := "createScript.sh"
	setupScript(url, script_name)

	ctx := context.Background()
	cli, err := dc.NewClientWithOpts(dc.FromEnv, dc.WithAPIVersionNegotiation())
	panicCheck(err)

	var email string
	fmt.Println(Yellow + "Please enter a unique email address." + Reset)
	fmt.Println(Green + "Enter an Email for new user." + Reset)
	fmt.Scanln(&email)
	email = setDefaultValue(email, "eln-user@kit.edu")

	passwd, fname, lname, abbreviation := userDataInput()

	// Set default values if left empty
	fname = setDefaultValue(fname, "ELN")
	lname = setDefaultValue(lname, "User")
	abbreviation = setDefaultValue(abbreviation, "CU1")

	/*
		//optionaly print to the console, but not necessarily required
		fmt.Println("--------------------------------------------------")
		fmt.Println(Yellow + "Following data will be send to the database:" + Reset)
		fmt.Println(Yellow+"email: ", email)
		fmt.Println(Yellow+"first name ", fname)
		fmt.Println(Yellow+"last name: ", lname)
		fmt.Println(Yellow+"abbreviation: ", abbreviation)
		fmt.Println(Reset + "--------------------------------------------------")
	*/

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
		fmt.Printf(Yellow+"Please enter a unique email and abbreviation. This error occured because either email: %s, or abbreviation:%s already exists.\n", arg1, arg5+Reset)
	} else {
		fmt.Println(Green + "Success" + Reset)
		fmt.Println(Green + "New user created successfully." + Reset)
	}
}

func handleUpdateUserLogic() {
	containerID := getContainerID_api(currentInstance, "eln")
	pathToScript := "/script/updateScript.sh"

	// Download a file and copy it inside running container
	url := "https://raw.githubusercontent.com/mehmood86/shellscripts/main/updateScript.sh"
	script_name := "updateScript.sh"
	setupScript(url, script_name)

	ctx := context.Background()
	cli, err := dc.NewClientWithOpts(dc.FromEnv, dc.WithAPIVersionNegotiation())
	panicCheck(err)

	var email string
	fmt.Println(Yellow + "Please enter a unique email address." + Reset)
	fmt.Println(Green + "Enter an Email for new user." + Reset)
	fmt.Scanln(&email)

	passwd, fname, lname, abbreviation := userDataInput()
	arg1 := email
	arg2 := passwd       //Password
	arg3 := fname        //first name
	arg4 := lname        //last name
	arg5 := abbreviation //abbreviation

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
		fmt.Printf(Yellow+"Please enter a unique email and abbreviation. This error occured because either email: %s, or abbreviation:%s already exists.\n", arg1, arg5+Reset)
	} else {
		fmt.Println(Green + "Success" + Reset)
		fmt.Println(Green + "User updated successfully." + Reset)
	}
}

func handleDeleteUserLogic() {
	fmt.Println("Handler is called.")
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
			fmt.Println("triggered by own call")
			handleDeleteUserLogic()
		} else {
			fmt.Println("triggered via cli menu")
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
