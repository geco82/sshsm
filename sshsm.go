package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/posener/complete"
	"github.com/tidwall/gjson"
)

func main() {

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	sessionsFilePath := fmt.Sprintf("%s/.config/sshsm", usr.HomeDir)

	if _, err := os.Stat(sessionsFilePath); os.IsNotExist(err) {
		newpath := filepath.Join(usr.HomeDir, ".config", "sshsm")
		os.MkdirAll(newpath, os.ModePerm)
	}

	sessionsFile := filepath.Join(sessionsFilePath, "sessions.json")

	if _, err := os.Stat(sessionsFile); os.IsNotExist(err) {
		os.Create(sessionsFile)
		defaultContent := []byte(`{ "sessions": []}`)
		ioutil.WriteFile(sessionsFile, defaultContent, 0644)
	}

	// Open our jsonFile
	jsonFile, err := os.Open(sessionsFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	sessionsResult := gjson.Get(string(byteValue), "sessions.#.id").Array()
	sessions := make([]string, len(sessionsResult))
	for i, v := range sessionsResult {
		sessions[i] = v.String()
	}

	// create the complete command
	cmp := complete.New(
		"sshsm",
		complete.Command{
			Sub: complete.Commands{
				// add a open sub command
				"open": {
					Args: complete.PredictFilesSet(sessions),
				},
			},
		},
	)

	// AddFlags adds the completion flags to the program flags,
	// in case of using non-default flag set, it is possible to pass
	// it as an argument.
	// it is possible to set custom flags open
	// so when one will type 'self -h', he will see '-complete' to install the
	// completion and -uncomplete to uninstall it.
	cmp.CLI.InstallName = "complete"
	cmp.CLI.UninstallName = "uncomplete"
	cmp.AddFlags(nil)

	// parse the flags - both the program's flags and the completion flags
	flag.Parse()

	// run the completion, in case that the completion was invoked
	// and ran as a completion script or handled a flag that passed
	// as argument, the Run method will return true,
	// in that case, our program have nothing to do and should return.
	if cmp.Complete() {
		return
	}

	// if the completion did not do anything, we can run our program logic here.
	if len(flag.Args()) == 2 && flag.Args()[0] == "open" {
		session := flag.Args()[1]
		sessionIndex := indexOf(session, sessions)
		if sessionIndex >= 0 {
			fmt.Printf("Opening %s...", session)
			sessionIndexString := fmt.Sprintf("sessions.%d", sessionIndex)

			sessionDetails := gjson.Get(string(byteValue), sessionIndexString)
			host := sessionDetails.Get("host").String()
			if host == "" {
				os.Exit(1)
			}
			user := sessionDetails.Get("user").String()
			if user != "" {
				host = fmt.Sprintf("%s@%s", user, host)
			}
			password := sessionDetails.Get("password").String()
			port := sessionDetails.Get("port").String()
			if port == "" {
				port = "22"
			}
			args := []string{"ssh", host, "-p", port}
			fmt.Println(args)

			fmt.Printf("\n%s\n", password)
			binary, lookErr := exec.LookPath("ssh")
			if lookErr != nil {
				panic(lookErr)
			}
			env := os.Environ()
			execErr := syscall.Exec(binary, args, env)
			if execErr != nil {
				panic(execErr)
			}
		}
		os.Exit(0)
	}
	os.Exit(0)
}

func indexOf(word string, data []string) int {
	for k, v := range data {
		if word == v {
			return k
		}
	}
	return -1
}
