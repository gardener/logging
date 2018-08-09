package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	OK    = "200 OK"
	HTTP  = "http://"
	HTTPS = "https://"
)

type ElasticSearch struct {
	nodes   []Node
	indices []Index
	url     string
}

type Node struct {
	FileSystem FileSystem `json:"fs"`
}

type FileSystem struct {
	Total FileSystemTotal `json:"total"`
}

type FileSystemTotal struct {
	TotalInBytes     int64 `json:"total_in_bytes"`
	FreeInBytes      int64 `json:"free_in_bytes"`
	AvailableInBytes int64 `json:"available_in_bytes"`
}

type Index struct {
	CreationDate int64  `json:"creation_date"`
	ProvidedName string `json:"provided_name"`
}

type Indices []Index

func (indices Indices) Len() int           { return len(indices) }
func (indices Indices) Swap(i, j int)      { indices[i], indices[j] = indices[j], indices[i] }
func (indices Indices) Less(i, j int) bool { return indices[i].CreationDate < indices[j].CreationDate }

func getURLWhitShemaString(url string) string {
	if !strings.HasPrefix(url, HTTP) {
		if strings.HasPrefix(url, HTTPS) {
			url = strings.Replace(url, HTTPS, HTTP, 1)
		} else {
			url = HTTP + url
		}
	}
	return url
}

func setHeaders(request *http.Request, headers map[string][]string) {
	for header, values := range headers {
		for _, value := range values {
			request.Header.Set(header, value)
		}
	}
}

func setForm(request *http.Request, form map[string]string) {
	for key, value := range form {
		request.Form.Add(key, value)
	}
}

func MakeHTTPRequest(method, url string, body []byte, headers map[string][]string, form map[string]string) (string, []byte, error) {
	url = getURLWhitShemaString(url)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return "", nil, err
	}
	setHeaders(req, headers)
	setForm(req, form)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.Status, nil, err
	}
	return resp.Status, responseBody, nil
}

func MakeGetHTTPrequest(url string) (string, []byte, error) {
	return MakeHTTPRequest("GET", url, nil, nil, nil)
}

func MakeDeleteHTTPrequest(url string) (string, []byte, error) {
	return MakeHTTPRequest("DELETE", url, nil, nil, nil)
}

func (e *ElasticSearch) GetIndices() []Index {
	indices := make([]Index, len(e.indices))
	copy(indices, e.indices)
	return indices
}

func (e *ElasticSearch) GetNode(index int) (Node, error) {
	if index >= 0 && index < len(e.nodes) {
		return e.nodes[index], nil
	}
	return Node{}, errors.New("index out of bound")
}

func (e *ElasticSearch) DeleteIndex(name string) error {
	url := path.Join(e.url, "/"+name)
	status, body, err := MakeDeleteHTTPrequest(url)
	if err != nil {
		return err
	} else if status != OK {
		return errors.New("delete index" + name + " status: " + status + "/n" + string(body))
	}
	log.Printf("Index %s has been deleted!\n", name)
	return nil
}

func (e *ElasticSearch) getIndicesNames() ([]string, error) {
	url := path.Join(e.url, "/_cat/indices?format=json")
	status, body, err := MakeGetHTTPrequest(url)
	if err != nil {
		return nil, err
	} else if status != OK {
		return nil, errors.New("Return status from getIndicesNames: " + status)
	}

	type Indices []struct {
		Health       string `json:"health"`
		Status       string `json:"status"`
		Index        string `json:"index"`
		UUID         string `json:"uuid"`
		Pri          string `json:"pri"`
		Rep          string `json:"rep"`
		DocsCount    string `json:"docs.count"`
		DocsDeleted  string `json:"docs.deleted"`
		StoreSize    string `json:"store.size"`
		PriStoreSize string `json:"pri.store.size"`
	}

	indices := new(Indices)
	err = json.Unmarshal(body, indices)
	if err != nil {
		return nil, errors.New("In getIndicesNames there was a problem with unmurshaling response body: " + err.Error())
	}

	indicesNames := make([]string, 0, len(*indices))
	for _, index := range *indices {
		indicesNames = append(indicesNames, index.Index)
	}

	return indicesNames, nil
}

func (e *ElasticSearch) getIndex(indexName string) (*Index, error) {
	url := path.Join(e.url, "/"+indexName+"/_settings")
	status, body, err := MakeGetHTTPrequest(url)
	if err != nil {
		return nil, err
	} else if status != OK {
		return nil, errors.New("Return status from extactNode: " + status)
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	rawIndexMap, ok := data[indexName].(map[string]interface{})["settings"].(map[string]interface{})["index"].(map[string]interface{})
	if !ok {
		return nil, errors.New("can't extract raw index map for index: " + indexName)
	}
	return getIndexFromRawMap(rawIndexMap)
}

func getIndexFromRawMap(rawIndexMap map[string]interface{}) (*Index, error) {
	if rawIndexMap == nil {
		return nil, errors.New("Nil rawIndexMap")
	}

	providedName, ok := rawIndexMap["provided_name"].(string)
	if !ok {
		return nil, errors.New("can't extract provided name from index")
	}

	creationDateStr, ok := rawIndexMap["creation_date"].(string)
	if !ok {
		return nil, errors.New("can't extract creating date from index: " + providedName)
	}

	creationDate, err := strconv.ParseInt(creationDateStr, 10, 64)
	if err != nil {
		return nil, err
	}

	return &Index{
		ProvidedName: providedName,
		CreationDate: creationDate,
	}, nil

}

func (e *ElasticSearch) ExtractIndices() error {
	indicesNames, err := e.getIndicesNames()
	if err != nil {
		return err
	}
	e.indices = make([]Index, 0, len(indicesNames))
	for _, indexName := range indicesNames {
		index, err := e.getIndex(indexName)
		if err != nil {
			return err
		}
		e.indices = append(e.indices, *index)
	}
	return nil
}

func (e *ElasticSearch) Init(url string) {
	e.url = url
}

func (e *ElasticSearch) ExtractNodes() error {
	url := path.Join(e.url, "/_nodes/stats/fs")
	status, body, err := MakeGetHTTPrequest(url)
	if err != nil {
		return err
	} else if status != OK {
		return errors.New("Return status from extactNode: " + status)
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(body, &data)
	if err != nil {
		return errors.New("In extractNode there was a problem with unmurshaling response body: " + err.Error())
	}

	data, ok := data["nodes"].(map[string]interface{})
	if !ok {
		return errors.New("Error extracting row nodes")
	}
	e.nodes = make([]Node, 0, len(data))
	for rawNodeName, rawNodeInterface := range data {
		rawNodeMap, ok := rawNodeInterface.(map[string]interface{})
		if !ok {
			return errors.New("Can't extract raw node map for " + rawNodeName)
		}
		node, err := getNode(rawNodeMap)
		if err != nil {
			return err
		}
		e.nodes = append(e.nodes, *node)
	}
	return nil
}

func getNode(rawNodeMap map[string]interface{}) (*Node, error) {
	rawFileSystemMap, ok := rawNodeMap["fs"].(map[string]interface{})
	if !ok {
		return nil, errors.New("Can't extract file system for node")
	}
	fileSystem, err := getFileSystem(rawFileSystemMap)
	if err != nil {
		return nil, err
	}
	return &Node{
		FileSystem: *fileSystem,
	}, nil
}

func getFileSystem(rawFileSystem map[string]interface{}) (*FileSystem, error) {
	rawFileSystemTotalMap, ok := rawFileSystem["total"].(map[string]interface{})
	if !ok {
		return nil, errors.New("Can't extract tatal from file system")
	}
	total, err := getFileSystemTotal(rawFileSystemTotalMap)
	if err != nil {
		return nil, err
	}
	return &FileSystem{
		Total: *total,
	}, nil
}

func getFileSystemTotal(rawFileSystemTotalMap map[string]interface{}) (*FileSystemTotal, error) {
	totalInBytes, ok := rawFileSystemTotalMap["total_in_bytes"].(float64)
	if !ok {
		return nil, errors.New("Can't extract total in bytes for node")
	}
	freeInBytes, ok := rawFileSystemTotalMap["free_in_bytes"].(float64)
	if !ok {
		return nil, errors.New("Can't extract free in bytes for node")
	}
	availableInBytes, ok := rawFileSystemTotalMap["available_in_bytes"].(float64)
	if !ok {
		return nil, errors.New("Can't extract available in bytes for node")
	}

	return &FileSystemTotal{
		TotalInBytes:     int64(totalInBytes),
		FreeInBytes:      int64(freeInBytes),
		AvailableInBytes: int64(availableInBytes),
	}, nil
}

func getBytesLeft(cluster *ElasticSearch) (int64, error) {
	cluster.ExtractNodes()
	node, err := cluster.GetNode(0)
	if err != nil {
		return 0, errors.New("no node avialable")
	}
	return node.FileSystem.Total.AvailableInBytes, nil
}

func removeOldestIndex(cluster *ElasticSearch) error {
	cluster.ExtractIndices()
	indices := cluster.GetIndices()
	if len(indices) < 1 {
		return errors.New("there are no indicies")
	}
	sort.Sort(Indices(indices))
	indexName := indices[0].ProvidedName
	return cluster.DeleteIndex(indexName)
}

func convertIntervaceKeyMapToStringKeyOne(data map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		strKey := fmt.Sprintf("%v", key)
		result[strKey] = value
	}
	return result
}

func converSliceOfInterfacesToSliceOfStrings(slice []interface{}) []string {
	result := make([]string, len(slice))
	for index, value := range slice {
		strValue := fmt.Sprintf("%v", value)
		result[index] = strValue
	}
	return result
}

// client:
//       hosts:
//         - elasticsearch-logging.garden.svc
//       port: 9200
func getEsApiFromConf(filename string) (string, error) {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	data := make(map[string]interface{})
	err = yaml.Unmarshal(yamlFile, data)
	if err != nil {
		return "", err
	}

	rawClient, ok := data["client"].(map[interface{}]interface{})
	if !ok {
		return "", errors.New("can't find client section in config file")
	}
	client := convertIntervaceKeyMapToStringKeyOne(rawClient)

	rawHosts, ok := client["hosts"].([]interface{})
	if !ok {
		return "", errors.New("can't find hosts section in client section in config file")
	}

	hosts := converSliceOfInterfacesToSliceOfStrings(rawHosts)
	if len(hosts) < 1 {
		return "", errors.New("empty hosts section in client section in config file")
	}

	rawPort, ok := client["port"].(int)
	if !ok {
		return "", errors.New("can't find port section in client section in config file")
	}
	port := strconv.Itoa(int(rawPort))
	return hosts[0] + ":" + port, nil
}

func main() {
	diskSpaceThreshold := flag.Int64("disk_threshold", 100000000, "The minimum maximum disk space left before deletion of the index")
	esAPI := flag.String("es_api", "", "The elasticsearch API")
	config := flag.String("config", "/etc/config/config.yml", "The config file for the curator")
	flag.Parse()
	var err error
	if *esAPI == "" {
		*esAPI, err = getEsApiFromConf(*config)
		if err != nil {
			log.Println(err.Error())
		}
		if *esAPI == "" {
			*esAPI = "localhost:9200"
		}
	}

	cluster := &ElasticSearch{}
	cluster.Init(*esAPI)
	bytesLeft, err := getBytesLeft(cluster)
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Printf("available bytes: %d", bytesLeft)
	for bytesLeft < *diskSpaceThreshold {
		log.Printf("bytes left: %d\nNeed: %d\n", bytesLeft, *diskSpaceThreshold)
		err = removeOldestIndex(cluster)
		if err != nil {
			log.Println(err.Error())
			return
		}
		time.Sleep(time.Duration(5) * time.Second)
		bytesLeft, err = getBytesLeft(cluster)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
}
