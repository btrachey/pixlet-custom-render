package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"tidbyt.dev/pixlet/encode"
	"tidbyt.dev/pixlet/runtime"
)

const (
	TidbytAPIPush = "https://api.tidbyt.com/v0/devices/%s/push"
	APITokenEnv   = "TIDBYT_API_TOKEN"
	DeviceIdEnv   = "TIDBYT_DEVICE_ID"
)

var (
	apiToken       string
	deviceId       string
	installationID string
	background     bool
)

type TidbytPushJSON struct {
	DeviceID       string `json:"deviceID"`
	Image          string `json:"image"`
	InstallationID string `json:"installationID"`
	Background     bool   `json:"background"`
}

func starfile(content string) []byte {
	starString :=
		"load(\"render.star\", \"render\")\n" +
		"def main(config):\n" +
		"  return render.Root(\n" +
		"    child = render.Box(\n" +
		"      render.WrappedText(\n" +
		"        content = \"" + content + "\"\n" +
		"      )\n" +
		"    )\n" +
		"  )\n"
	return []byte(starString)
}

func doPost(imgContent string) bool {
	apiToken = os.Getenv(APITokenEnv)
	deviceId = os.Getenv(DeviceIdEnv)
	payload, err := json.Marshal(
		TidbytPushJSON{
			DeviceID:       deviceId,
			Image:          imgContent,
			InstallationID: installationID,
			Background:     background,
		})
	if err != nil {
		log.Printf("failed to marchal json: %v\n", err)
		return false
	}

	client := &http.Client{}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf(TidbytAPIPush, deviceId),
		bytes.NewReader(payload))
	if err != nil {
		log.Printf("creating POST request: %v\n", err)
		return false
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiToken))

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("pushing to API: %v\n", err)
		return false
	}

	if resp.StatusCode != 200 {
		log.Printf("Tidbyt API returned status %s\n", resp.Status)
		body, _ := ioutil.ReadAll(resp.Body)
		log.Println(string(body))
		return false
	}

	return true
}

func imageGen(content string) string {
	magnify := 1
	script := "test.star"
	src := starfile(content) // max 54 chars
	applet := runtime.Applet{}
	err := applet.Load(script, src, nil)
	if err != nil {
		log.Printf("Unable to load scriptfile %v\n", err)
	}
	config := map[string]string{}
	roots, err := applet.Run(config)
	if err != nil {
		log.Printf("Error running script: %s\n", err)
		os.Exit(1)
	}
	screens := encode.ScreensFromRoots(roots)

	filter := func(input image.Image) (image.Image, error) {
		if magnify <= 1 {
			return input, nil
		}
		in, ok := input.(*image.RGBA)
		if !ok {
			return nil, fmt.Errorf("image not RGBA, very weird")
		}

		out := image.NewRGBA(
			image.Rect(
				0, 0,
				in.Bounds().Dx()*magnify,
				in.Bounds().Dy()*magnify),
		)
		for x := 0; x < in.Bounds().Dx(); x++ {
			for y := 0; y < in.Bounds().Dy(); y++ {
				for xx := 0; xx < 10; xx++ {
					for yy := 0; yy < 10; yy++ {
						out.SetRGBA(
							x*magnify+xx,
							y*magnify+yy,
							in.RGBAAt(x, y),
						)
					}
				}
			}
		}

		return out, nil
	}

	var buf []byte
	buf, err = screens.EncodeWebP(filter)
	if err != nil {
		log.Printf("Unable to encode to webp %v\n", err)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func main() {
	encodedImgString := imageGen("This is some test content")
	doPost(encodedImgString)
}
