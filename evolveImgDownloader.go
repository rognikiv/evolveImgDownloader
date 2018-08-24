/*Author: Ives Nikiema*/

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
)

var (
	reVehicleDetail = regexp.MustCompile("(?:<a href=\")(/trucks/.+?)(?:\">)")
	reStockNumber   = regexp.MustCompile("vehicle_stock=(.+?)&")
	rePhotoLink     = regexp.MustCompile("<a href=\"(.+?)\" class=\"openphoto\">")
	wg              = sync.WaitGroup{}
	counter         = new(downloadCounter)
)

type downloadCounter struct {
	images int
	units  int
	sync.Mutex
}

func (c *downloadCounter) incImg() {
	c.Lock()
	c.images++
	c.Unlock()
}

func (c *downloadCounter) incUnit(delta int) {
	c.Lock()
	c.units += delta
	c.Unlock()
}

func main() {
	var (
		homeUri string //http://inventory.westernisuzutruck.com/
		homeDir string
	)

	fmt.Print("Enter SRP Url: ")
	fmt.Scan(&homeUri)
	fmt.Print("Dir (optional): ")
	fmt.Scan(&homeDir)
	if !strings.Contains(homeUri, "http://") {
		homeUri = "http://" + homeUri
	}
	baseUri, err := url.Parse(homeUri)
	resp, err := http.Get(baseUri.String())
	if err != nil {
		log.Fatalf("Error: Unable to access %s\n\tError msg: %s\n\n\tError code: %x\n", homeUri, err, 0)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)

	pwd, err := os.Getwd()
	pwd = strings.Replace(pwd, "\\", "/", -1)
	homeDir = path.Join(pwd, homeDir)
	fmt.Println("Download Directory: ", homeDir)
	for _, match := range reVehicleDetail.FindAllSubmatch(content, -1) {
		uri, err := url.Parse(fmt.Sprintf("%s", match[1]))
		if err != nil {
			fmt.Printf("Error parsing %s\n\tError msg: %s\n\tError code: %x\n", fmt.Sprintf("%s", match[1]), err, 1)
			continue
		}
		uri = baseUri.ResolveReference(uri)
		wg.Add(1)
		go getImages(uri, homeDir)
		counter.incUnit(1)
	}
	wg.Wait()
	fmt.Printf("\nDownloaded: %d Photos from %d vehicle units\n", counter.images, counter.units)
	fmt.Printf("*** All downloads completed! ***")
}

func getImages(uri *url.URL, homeDir string) {
	resp, err := http.Get(uri.String())
	if err != nil {
		fmt.Printf("Error: Unable to access %s\n\tError msg: %s\n\tError code: %x\n", uri, err, 2)
		return
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)

	stk := fmt.Sprintf("%s", reStockNumber.FindSubmatch(content)[1])
	relativePath := path.Join(homeDir, stk)
	os.MkdirAll(relativePath, os.ModeDir)
	//imageUris := make([]*url.URL, 0)
	for _, match := range rePhotoLink.FindAllSubmatch(content, -1) {
		imgUri, err := url.Parse(fmt.Sprintf("http:%s", match[1]))
		filename := path.Base(imgUri.String())
		if strings.Contains(filename, "nophoto.jpg") {
			fmt.Printf("FYI: Stock Number: %s has ZERO Photos\n", stk)
			counter.incUnit(-1)
			continue
		}
		if err != nil {
			fmt.Printf("Error parsing %s\n\tError msg: %s\n\tError code: %x\n", fmt.Sprintf("%s", match[1]), err, 3)
			continue
		}
		//imageUris = append(imageUris, imgUri)
		imgResp, err := http.Get(imgUri.String())
		if err != nil {
			fmt.Printf("Unable to download %s\n\tError MSG: %s\n\tError code: %x\n", imgUri, err, 4)
			continue
		}
		imgPath := path.Join(relativePath, filename)
		f, err := os.Create(imgPath)
		if err != nil {
			fmt.Printf("Error creating file: %s\n\tError msg: %s\n\tError code: %x\n", imgPath, err, 5)
			continue
		}

		n, err := io.Copy(f, imgResp.Body)
		if err != nil {
			fmt.Printf("Error: unable to download img %s\n\tError msg:%s\n\tError code: %x\n", imgUri.String(), err, 6)
			continue
		}
		f.Close()
		imgResp.Body.Close()
		fmt.Printf("Downloaded %s (%d Kb)\n", imgUri.String(), n/1024)
		counter.incImg()
	}
	wg.Done()
}
