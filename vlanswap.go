package main
import (
_       "github.com/go-sql-driver/mysql"
		"database/sql"
		"bytes"
		"golang.org/x/crypto/ssh"
        "fmt"
		"log"
		"strconv"
		"os"
		"bufio"
		"strings"
)

const (
	DB_IPADDR			= ""
	DB_PORT				= ""
	DB_USERNAME 		= ""
	DB_PASSWORD 		= ""
	DB_NAME				= ""
	NETSWITCH_USERNAME	= ""
	NETSWITCH_PASSWORD	= ""
)

func removeDuplicateValues(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list,entry)
		}
	}
	return list
}

func portCount(task_id int, switch_fqdn string) int {
	var count int

	db, err := sql.Open("mysql", DB_USERNAME + ":" + DB_PASSWORD + "@tcp(" + DB_IPADDR + ":" + DB_PORT + ")/" + DB_NAME)
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()

	row := db.QueryRow("SELECT COUNT(*) FROM networkassistant_task JOIN networkassistant_port ON networkassistant_task.port_id = networkassistant_port.port_id JOIN networkassistant_switch ON networkassistant_port.switch_fqdn_id = networkassistant_switch.id WHERE request_id = ? AND fqdn = ?", task_id, switch_fqdn)
	err = row.Scan(&count)
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()

	return count
}

func portList(task_id int, switch_fqdn string) []string {
	ports := make([]string, 0) 	// Create the Slice variable where the switch ports will be placed.
	var port string 			// Define the String variable that the port will be assigned to while being placed into the Slice via for loop.

	db, err := sql.Open("mysql", DB_USERNAME + ":" + DB_PASSWORD + "@tcp(" + DB_IPADDR + ":" + DB_PORT + ")/" + DB_NAME)
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT networkassistant_port.switch_port FROM networkassistant_task JOIN networkassistant_port ON networkassistant_task.port_id = networkassistant_port.port_id JOIN networkassistant_switch ON networkassistant_port.switch_fqdn_id = networkassistant_switch.id WHERE request_id = ? AND fqdn = ?", task_id, switch_fqdn)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next(){
		err := rows.Scan(&port)
		if err != nil {
				log.Fatal(err)
		}
		ports = append(ports, port)		
	}
		err = rows.Err()
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()
	return ports
}

func switchList(task_id int) []string {
	netswitches := make([]string, 0) 	// Create the Slice variable where the switch ports will be placed.
	var netswitch string 			// Define the String variable that the port will be assigned to while being placed into the Slice via for loop.

	db, err := sql.Open("mysql", DB_USERNAME + ":" + DB_PASSWORD + "@tcp(" + DB_IPADDR + ":" + DB_PORT + ")/" + DB_NAME)
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT networkassistant_switch.fqdn FROM networkassistant_switch JOIN networkassistant_port ON networkassistant_switch.id = networkassistant_port.switch_fqdn_id JOIN networkassistant_task ON networkassistant_port.port_id = networkassistant_task.port_id WHERE request_id = ?", task_id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next(){
		err := rows.Scan(&netswitch)
		if err != nil {
				log.Fatal(err)
		}
		netswitches = append(netswitches, netswitch)		
	}
		err = rows.Err()
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()
	return netswitches
}

func changeCMDSet(netswitch string, port []string, vlan_number int) []string {
	cmd := make([]string, 0)

	cmd = append(cmd, "configure terminal")
	for x := 0; x < len(ports); x++ {
		cmd = append(cmd, "interface " + port[x])
		cmd = append(cmd, "switchport access vlan" + strconv.Itoa(vlan_number))
		cmd = append(cmd, "spanning-tree portfast")
		cmd = append(cmd, "show run interface " + port[x])
	}
	
	return cmd
}

func checkCMDSet(netswitch string, port []string) []string {
	cmd := make([]string, 0)
	
	for x := 0; x < len(ports); x++ {
		cmd = append(cmd, "show run interface " + port[x])
	}

	return cmd
}

func bondCMDSet(netswitch string, vlan_number int, portchannel int, port []string) []string {
	cmd := make([]string, 0)

	cmd = append(cmd, "configure terminal")
	cmd = append(cmd, "interface port-channel "+ strconv.Itoa(port-channel))
	cmd = append(cmd, "switchport")
	cmd = append(cmd, "switchport access vlan " + strconv.Itoa(vlan_number))
	for x := 0; x < len(ports); x++ {
		cmd = append(cmd, "interface " + port[x])
		cmd = append(cmd, "channel-group " + strconv.Itoa(port-channel) + " mode active")
		cmd = append(cmd, "no shut")
	}
	cmd = append(cmd, "interface port-channel "+ strconv.Itoa(port-channel))
	cmd = append(cmd, "no shut")
	
	return cmd
}

func sshProcedure(netswitch string, cmd []string) {
	// SSH Creds to log into remote device
	config := &ssh.ClientConfig{
		User:				NETSWITCH_USERNAME,
		Auth:				[]ssh.AuthMethod{ssh.Password(NETSWITCH_PASSWORD)},
		HostKeyCallback:	ssh.InsecureIgnoreHostKey(),
	}
	// Used for loop to run through command set, "_" throws away the key data while v represents the acutal command (value)
	for i := 0; i < len(cmd); i++ {
		// This is who the script will be logging into
		client, err := ssh.Dial("tcp", netswitch, config)
		if err != nil {
			log.Fatal("Failed to dial: ", err)
		}
		// This initiates the connection to the remote device
		session, err := client.NewSession()
		if err != nil {
			log.Fatal("Failed to create session: ", err)
		}
		defer session.Close()
		// Defines a variable to capture the output of the commands being run
		var cmd_output bytes.Buffer
		session.Stdout = &cmd_output
		fmt.Printf("**** %s *** ", cmd[i])
		// This is where the commands actually being sent to the remote device
		session.Run(cmd[i])
		// Prints output to screen
		fmt.Println(cmd_output.String())
		fmt.Printf("#########\n")

	}
}
	
func verifyChangeConfiguration(output interface{}, vlan int) {
	// Validate the command was applied.
	scanner := bufio.NewScanner(&cmd_output)
	line := 1
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "switchport access vlan " + strconv.Itoa(vlan))  {
			fmt.Println("True")
		} else {
			fmt.Println("False")
		}
		line++
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}	

func singleQueryLookup(task_id int, lookupItem string) string {
	var output string
	// Connect to the Database
	db, err := sql.Open("mysql", DB_USERNAME + ":" + DB_PASSWORD + "@tcp(" + DB_IPADDR + ":" + DB_PORT + ")/" + DB_NAME)
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()
	// Query Database for results
	switch lookupItem {
	case "request_type":
		rows, err := db.Query("SELECT networkassistant_task.request_type FROM networkassistant_task WHERE request_id = ?", task_id)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next(){
			err := rows.Scan(&output)
			if err != nil {
					log.Fatal(err)
			}
		}
			err = rows.Err()
		if err != nil {
				log.Fatal(err)
		}
		defer db.Close()
	case "vlan_number":
		rows, err := db.Query("SELECT networkassistant_vlan.number FROM networkassistant_task JOIN networkassistant_vlan ON networkassistant_task.vlan_name = networkassistant_vlan.name WHERE request_id = ?", task_id)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next(){
			err := rows.Scan(&output)
			if err != nil {
					log.Fatal(err)
			}
		}
			err = rows.Err()
		if err != nil {
				log.Fatal(err)
		}
		defer db.Close()
	case "requester":
		rows, err := db.Query("SELECT auth_user.username FROM networkassistant_task JOIN auth_user ON networkassistant_task.requester_id = auth_user.id WHERE request_id = ?", task_id)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next(){
			err := rows.Scan(&output)
			if err != nil {
					log.Fatal(err)
			}
		}
			err = rows.Err()
		if err != nil {
				log.Fatal(err)
		}
		defer db.Close()
	}
	return output
}

func updatePortDBEntry(netswitch string, port string, vlan_number int){ // Update the DB to reflect the change is the port configuration
	netswitches := make([]string, 0) 	// Create the Slice variable where the switch ports will be placed.
	var netswitch string 				// Define the String variable that the port will be assigned to while being placed into the Slice via for loop.

	db, err := sql.Open("mysql", DB_USERNAME + ":" + DB_PASSWORD + "@tcp(" + DB_IPADDR + ":" + DB_PORT + ")/" + DB_NAME)
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()

	result, err := db.Exec("UPDATE networkassistant_port JOIN networkassistant_switch ON networkassistant_port.switch_fqdn_id = networkassistant_switch.id JOIN networkassistant_vlan ON networkassistant_port.vlan_name_id = networkassistant_vlan.id SET networkassistant_port.vlan_name_id = ? WHERE networkassistant_switch.fqdn = ?, networkassistant_port.port_id = ?, networkassistant_vlan.name = ?", vlan_number, netswitch, port, vlan_number)
	if err != nil {
		log.Fatal(err)
	}
	
}

func updateTaskDBEntry(task_id int, netswitch string, port string, status_code string, status_detail string) { // Update the task status (by port) to provide live feed to the front-end
	// DB Connection
	db, err := sql.Open("mysql", DB_USERNAME + ":" + DB_PASSWORD + "@tcp(" + DB_IPADDR + ":" + DB_PORT + ")/" + DB_NAME)
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()
	// CMD Against DB
	rows, err := db.Query("UPDATE networkassistant_task JOIN networkassistant_port ON networkassistant_task.port_id = networkassistant_port.port_id JOIN networkassistant_switch ON networkassistant_port.switch_fqdn_id = networkassistant_switch.id SET status_id = ?, status = ? WHERE request_id = ?, networkassistant_switch.fqdn = ?, networkassistant_port.switch_port", status_code, status_detail, task_id, netswitch, port)
	if err != nil {
			log.Fatal(err)
	}
	defer rows.Close()
	var numDelete int
	for rows.Next() {
		numDeleted += 1
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func deleteDBTask(task_id int) { // Delete the Task entry once everything is complete.
	// DB Connection
	db, err := sql.Open("mysql", DB_USERNAME + ":" + DB_PASSWORD + "@tcp(" + DB_IPADDR + ":" + DB_PORT + ")/" + DB_NAME)
	if err != nil {
			log.Fatal(err)
	}
	defer db.Close()
	// CMD Against DB
	rows, err := db.Query("DELETE FROM networkassistant_task WHERE request_id = ?", task_id)
	if err != nil {
			log.Fatal(err)
	}
	defer rows.Close()
	var numDelete int
	for rows.Next() {
		numDeleted += 1
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	task_id := os.Args[0]												// The Task ID generated by Django and is assoicated in the database.
	request_type := singleQueryLookup(1, "request_type")				// Check the request type (change/check/bond) based on the Task ID in the Django database.
	vlan_number := singleQueryLookup(1, "vlan_number")					// Check the VLAN number based on the Task ID in the Django database.
	requester := singleQueryLookup(1, "requester")						// Check who the requester is based on the Task ID in the Django database.
	netswitch := removeDuplicateValues(switchList(1))					// Retrieve the switches that are involved on the request.
	port_speed := "N/A"
	port_duplex := "N/A"
	port_status := "N/A"
	switch request_type {
	case "bond":
		if len(netswitch) == 1 {
			ports := portList(task_id, netswitch[i])						// Gather the ports that'll be configured for this switch.
			cmd := bondCMDSet(request_type, netswitch[i], vlan_number, ports)	// Generate command set that'll be ran on the switches
			sshProcedure(netswitch[i], cmd)
			results := verifyConfiguration(netswitch, ports)
		} else {

		}
	case "change":
		for i := 0; i < len(netswitch); i++ { 								// Loop through the switches running the switch command set.
			ports := portList(task_id, netswitch[i])						// Gather the ports that'll be configured for this switch.
			cmd := changeCMDSet(request_type, netswitch[i], vlan_number, ports)	// Generate command set that'll be ran on the switches
			sshProcedure(netswitch[i], cmd)
			results := verifyConfiguration(netswitch, ports)
	
		}
	case "check":
		for i := 0; i < len(netswitch); i++ { 								// Loop through the switches running the switch command set.
			ports := portList(task_id, netswitch[i])						// Gather the ports that'll be configured for this switch.
			cmd := checkCMDSet(request_type, netswitch[i], vlan_number, ports)	// Generate command set that'll be ran on the switches
			sshProcedure(netswitch[i], cmd)
			results := verifyConfiguration(netswitch, ports)
	
		}
	}
}
