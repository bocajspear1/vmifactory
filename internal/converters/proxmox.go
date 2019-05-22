package converters

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type ProxmoxConfig struct {
	Host        string `json:"host"`
	Node        string `json:"node"`
	Username    string `json:"username"`
	SSLIgnore   bool   `json:"ssl_ignore_invalid"`
	Password    string `json:"password"`
	StorageMove string `json:"storage_move"`
}

type proxmoxAuthData struct {
	Data struct {
		Ticket              string `json:"ticket"`
		Username            string `json:"username"`
		Capabilities        int    `json:"-"`
		CSRFPreventionToken string `json:"CSRFPreventionToken"`
	}
}

type proxmoxSimpleData struct {
	Data map[string]string `json:"data"`
}

type proxmoxStringData struct {
	Data string `json:"data"`
}

func loadConfig(path string) (*ProxmoxConfig, error) {
	configFile, ferr := ioutil.ReadFile(path)
	if ferr != nil {
		return nil, errors.New("Could not parse Proxmox config file: File " + path + " not found")
	}

	var config ProxmoxConfig
	jerr := json.Unmarshal(configFile, &(config))
	if jerr != nil {
		return nil, jerr
	}

	return &config, nil
}

func doPost(url string, values url.Values, authToken string, csrfToken string) ([]byte, error) {

	netClient := &http.Client{
		Timeout: time.Second * 10,
	}

	req, rerr := http.NewRequest(http.MethodPost, url, strings.NewReader(values.Encode()))
	if rerr != nil {
		return nil, rerr
	}

	if csrfToken != "" {
		req.Header.Set("CSRFPreventionToken", csrfToken)
	}

	if authToken != "" {
		cookie := http.Cookie{}
		cookie.Name = "PVEAuthCookie"
		cookie.Value = authToken
		req.AddCookie(&cookie)
	}

	req.ParseForm()

	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))

	resp, err := netClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, rerr := ioutil.ReadAll(resp.Body)
	if rerr != nil {
		return nil, rerr
	}
	return body, nil
}

func ProxmoxRunVBoxConverter(targetName string, sourceFiles []string) error {

	config, err := loadConfig("./config/proxmox.json")
	if err != nil {
		return err
	}

	urlBase := "https://" + config.Host + "/api2/json/"

	if config.SSLIgnore {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Authenticate to the Proxmox Server
	vals := url.Values{}
	fmt.Println(config.Username)
	vals.Add("username", config.Username)
	vals.Add("password", config.Password)

	authResp, rerr := doPost(urlBase+"access/ticket", vals, "", "")
	if rerr != nil {
		return rerr
	}

	respJSON := proxmoxAuthData{}
	fmt.Println(string(authResp))
	jsonErr := json.Unmarshal(authResp, &respJSON)
	if jsonErr != nil {
		return jsonErr
	}

	crsfToken := respJSON.Data.CSRFPreventionToken
	fmt.Println(crsfToken)
	authToken := respJSON.Data.Ticket
	fmt.Println(authToken)

	execVals := url.Values{}
	// execVals.Add("cmd", "sleep 30")
	// execVals.Add("node", config.Node)

	execResp, rerr := doPost(urlBase+"nodes/"+config.Node+"/vncshell", execVals, authToken, crsfToken)
	if rerr != nil {
		return rerr
	}
	fmt.Println(string(execResp))
	// Tell the Proxmox server to download our image file
	// Get the current disk of our target VM
	// Set the disk of the target VM
	// Remove the old disk of the target VM
	// Move the disk
	return nil
}
