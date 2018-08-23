/*Author: Ives Nikiema*/

package main

import (
	"net/http"
	"log"
	"regexp"
	"io/ioutil"
	"fmt"
	"net/url"
	"sync"
	"path"
	"os"
	"strings"
)

var (
	reVehicleDetail = regexp.MustCompile("(?:<a href=\")(/trucks/.+?)(?:\">)")
	reStockNumber   = regexp.MustCompile("vehicle_stock=(.+?)&")
	rePhotoLink     = regexp.MustCompile("<a href=\"(.+?)\" class=\"openphoto\">")
	wg              = sync.WaitGroup{}
)

func main() {
	var (
		homeUri string //http://inventory.westernisuzutruck.com/
		homeDir string
	)

	fmt.Print("Enter SRP Url: ")
	fmt.Scan(&homeUri)
	fmt.Print("Dir (optional): ")
	fmt.Scan(&homeDir)

	baseUri, err := url.Parse(homeUri)
	resp, err := http.Get(baseUri.String())
	if err != nil {
		log.Fatalf("Error: Unable to access %s\n\tError msg: %s\n\n\tError code: %x\n", homeUri, err, 0)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)

	for _, match := range reVehicleDetail.FindAllSubmatch(content, -1) {
		uri, err := url.Parse(fmt.Sprintf("%s", match[1]))
		if err != nil {
			fmt.Printf("Error parsing %s\n\tError msg: %s\n\tError code: %x\n", fmt.Sprintf("%s", match[1]), err, 1)
		}
		uri = baseUri.ResolveReference(uri)
		wg.Add(1)
		go getImages(uri.String(), homeDir)
	}
	wg.Wait()
	fmt.Printf("All downloads completed!")
}

func getImages(uriString string, homeDir string) {
	uri, _ := url.Parse(uriString)
	resp, err := http.Get(uri.String())
	if err != nil {
		log.Fatalf("Error: Unable to access %s\n\tError msg: %s\n\tError code: %x\n", uri, err, 2)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)

	stk := fmt.Sprintf("%s", reStockNumber.FindSubmatch(content)[1])
	pwd, err := os.Getwd()
	relativePath := path.Join(pwd, homeDir, stk)
	os.MkdirAll(relativePath, os.ModeDir)
	//imageUris := make([]*url.URL, 0)
	for _, match := range rePhotoLink.FindAllSubmatch(content, -1) {
		imgUri, err := url.Parse(fmt.Sprintf("http:%s", match[1]))
		if err != nil {
			fmt.Printf("Error parsing %s\n\tError msg: %s\n\tError code: %x\n", fmt.Sprintf("%s", match[1]), err,3)
		}
		//imageUris = append(imageUris, imgUri)
		imgResp, err := http.Get(imgUri.String())
		if err != nil {
			fmt.Printf("Unable to download %s\n\tError MSG: %s\n\tError code: %x\n", imgUri, err, 4)
			continue
		}
		imgPath := path.Join(relativePath, path.Base(imgUri.String()))
		imgPath = strings.Replace(imgPath, "/", "\\", -1)
		f, err := os.Create(imgPath)
		if err != nil {
			fmt.Printf("Error creating file: %s\n\tError msg: %s\n\tError code: %x\n", imgPath, err, 5)
			continue
		}
		content, err := ioutil.ReadAll(imgResp.Body)
		if err != nil {
			fmt.Printf("Error: unable to download img %s\n\tError msg:%s\n\tError code: %x\n", imgUri.String(), err, 6)
			continue
		}
		_, err = f.Write(content)
		f.Close()
		imgResp.Body.Close()
		if err != nil {
			fmt.Printf("Unable to save url: %s to Director: %s\n\tError msg: %s\n\tError code: %x\n", imgUri.String(), imgPath, err, 7)
			continue
		}
		fmt.Printf("Downloaded %s -> %s\n", imgUri.String(), imgPath)
	}
	wg.Done()
}
