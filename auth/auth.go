package auth

import (
	"Mars/icarus"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
	"strconv"
)

var ServerInitDone = make(chan bool)

type Server struct {
	IP     string
	Port   int
	Nation string
	Client icarus.IcarusClient
}

var Servers []Server

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetFromConf(toSearchFor string) any {
	result, err := os.ReadFile("conf/conf.json")
	if err != nil {
		fmt.Println("[GFC] - Error reading conf.json - ", err.Error())
		os.Exit(1)
	}
	var confData any
	err = json.Unmarshal(result, &confData)
	if err != nil {
		fmt.Println("[GFC] - Error unmarshalling - ", err.Error())
		os.Exit(1)
	}
	temp := confData.(map[string]interface{})
	return temp[toSearchFor]
}

func GetServerDetails() {
	result, err := os.ReadFile("conf/conf.json")
	if err != nil {
		fmt.Println("[GSD] - Error reading conf.json - ", err.Error())
		os.Exit(1)
	}
	var confData any
	err = json.Unmarshal(result, &confData)
	if err != nil {
		fmt.Println("[GSD] - Error unmarshalling - ", err.Error())
		os.Exit(1)
	}
	fullJSON := confData.(map[string]interface{})
	for i := range len(fullJSON) {
		j := fullJSON[strconv.Itoa(i)]
		serverData, ok := j.(map[string]interface{})
		if !ok {
			fmt.Printf("[GSD] - Skipping invalid server data for key %d\n", i)
			break
		}
		ip, ok := serverData["ICARUS_SERVER_IP"].(string)
		if !ok {
			log.Printf("[GSD] - Invalid IP for server %d", i)
			os.Exit(1)
		}
		nation, ok := serverData["NATION"].(string)
		if !ok {
			log.Printf("[GSD] - Invalid Nation for server %d", i)
			os.Exit(1)
		}
		portFloat, ok := serverData["ICARUS_SERVER_PORT"].(float64) // JSON numbers are float64
		if !ok {
			log.Printf("[GSD] - Invalid Port for server %d", i)
			os.Exit(1)
		}
		port := int(portFloat)
		fmt.Printf("[GSD] - Checking if certs exists for %d\n", i)
		certsExists, err := exists(fmt.Sprintf("conf/certs/%d", i))
		if err != nil {
			log.Printf("[GSD] - Error checking if cert file exists %s", err.Error())
			os.Exit(1)
		}
		var icarusClient icarus.IcarusClient = nil
		if certsExists {
			fmt.Println("[GSD] - Found certs directory")
			icarusClient = CreateSecureIcarusClient(ip, fmt.Sprintf("%d", port), fmt.Sprintf("conf/certs/%d", i))
			fmt.Printf("[GSD] - Added %d server %s - %s:%d\n", i, nation, ip, port)
		} else {
			icarusClient = CreateIcarusClient(ip, fmt.Sprintf("%d", port))
			fmt.Printf("[GSD] - Added %d server %s - %s:%d\n", i, nation, ip, port)
		}
		Servers = append(Servers, Server{
			IP:     ip,
			Port:   port,
			Nation: nation,
			Client: icarusClient,
		})
	}
	ServerInitDone <- true
}

func CreateIcarusClient(SERVER_ADDR string, SERVER_PORT string) icarus.IcarusClient {
	icarusConnString := fmt.Sprintf("%s:%s", SERVER_ADDR, SERVER_PORT)
	icarusConn, err := grpc.NewClient(icarusConnString, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("grpc.NewClient(\"%s\") failed: %s\n", icarusConnString, err)
		return nil
	}
	icarusClient := icarus.NewIcarusClient(icarusConn)
	return icarusClient
}

func CreateSecureIcarusClient(SERVER_ADDR string, SERVER_PORT string, CERT_DIR string) icarus.IcarusClient {
	caCrt := fmt.Sprintf("%s/ca.crt", CERT_DIR)
	icarusCrt := fmt.Sprintf("%s/icarus.crt", CERT_DIR)
	icarusKey := fmt.Sprintf("%s/icarus.key", CERT_DIR)

	certificate, err := tls.LoadX509KeyPair(icarusCrt, icarusKey)
	if err != nil {
		fmt.Printf("[CSIC] - Failed to load key pair for %s: %s\n", SERVER_ADDR, err)
		return nil
	}

	ca, err := os.ReadFile(caCrt)
	if err != nil {
		fmt.Printf("[CSIC] - Failed to read CA certificate for %s: %s\n", SERVER_ADDR, err)
		return nil
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(ca) {
		fmt.Printf("[CSIC] - Failed to append the CA certificate to CA pool for %s: %s\n", SERVER_ADDR, err)
		return nil
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		RootCAs:      caPool,
	}

	icarusConnString := fmt.Sprintf("%s:%s", SERVER_ADDR, SERVER_PORT)
	icarusConn, err := grpc.NewClient(icarusConnString, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		fmt.Printf("[CSIC] - grpc.NewClient(\"%s\") failed: %s\n", icarusConnString, err)
		return nil
	}
	icarusClient := icarus.NewIcarusClient(icarusConn)
	return icarusClient

	/*tlsConfig := &tls.Config{
		ServerName:   ICARUS_ADDRESS,
		Certificates: []tls.Certificate{cert},
		RootCAs:      ca,
	}

	IcarusConnString := fmt.Sprintf("%s:%s", config.IcarusAddress, config.IcarusPort)
	IcarusConn, err := grpc.Dial(IcarusConnString, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	if err != nil {
		fmt.Println(err)
		slog.Error("Drone controller failed to connect to Icarus")
		os.Exit(1)
	}
	client := icarus.NewIcarusClient(IcarusConn)

	return client*/
}
