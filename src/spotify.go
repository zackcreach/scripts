package main

import (
	"fmt"
	"net/http"
	"net/url"
  "io"
  "io/ioutil"
  "path/filepath"
  "log"
	"flag"
	"os"
  "strings"
  "regexp"
  "encoding/json"
)

var SPOTIFY_TOKEN_FILEPATH string = filepath.Join(os.Getenv("HOME"), "/Library/Preferences/Spotify")
const SPOTIFY_TOKEN_FILENAME string = "spotify_access_token.txt"
const HOST string = "https://api.spotify.com/v1/me/player"
const VALUE_DEFAULT int = 5

var action string
var option string
var value int

type RefreshTokenResponse struct {
  AccessToken string `json:"access_token"`
  TokenType string `json:"token_type"`
  ExpiresIn int `json:"expires_in"`
  Scope string `json:"scope"`
}

type UnauthorizedResponse struct {
   Status int `json:"status"`
}

type CurrentlyPlayingResponse struct {
  IsPlaying bool `json:"is_playing"`
}

type DevicesResponse struct {
  Name bool `json:"name"`
  IsPlaying bool `json:"name.is_playing"`
}

func init() {
	flag.StringVar(&action, "action", "", "Action flag such as 'play'")
	flag.StringVar(&option, "option", "", "Option flag to specify the device, for example")
	flag.IntVar(&value, "value", 5, "Value flag corresponding to the option (e.g. 10)")
	flag.Parse()
}

func request(method string, path string, options io.Reader, headers map[string]string) []byte {
  client := &http.Client{}

  var endpoint string

  found, _ := regexp.MatchString("^/", path)

  if found {
    endpoint = HOST + path
  } else {
    endpoint = path
  }

	req, _ := http.NewRequest(method, endpoint, options)

  // Add passed in headers to request, otherwise assume auth header is needed
  if headers != nil {
    for key, value := range headers {
      req.Header.Add(key, value)
    }
  } else {
    file := filepath.Join(SPOTIFY_TOKEN_FILEPATH, SPOTIFY_TOKEN_FILENAME)
    tokenInFile, err := ioutil.ReadFile(file)

    if err != nil {
      log.Print("Error: Failed to read token from file: ", err)

      if err := os.MkdirAll(SPOTIFY_TOKEN_FILEPATH, os.ModePerm); err != nil {
        log.Print("Error: Failed to create directory: ", err)
      }

      if err := ioutil.WriteFile(file, []byte("PLACEHOLDER"), os.ModePerm); err != nil {
        log.Print("Error: Failed to write file: ", err)
      }
    }
    
    var accessToken string = strings.ReplaceAll(string(tokenInFile), "\n", "")
    req.Header.Add("Authorization", "Bearer " + accessToken)
  }

  res, err := client.Do(req)

  // Handle general request errors
  if err != nil {
    log.Print("Error: Failed to request url: ", endpoint, err)
    return nil
  }

  // Handle 401 error (token refresh needed)
  if res.StatusCode == 401 {
    updateToken(method, path, options, headers)
    return nil
  }

  bytes, err := ioutil.ReadAll(res.Body)

  // Handle bytes (body) read errors
  if err != nil {
    log.Print("Error: Failed to read bytes: ", err)
  }

  defer res.Body.Close()
  return bytes
}

func updateToken(method string, path string, options io.Reader, headers map[string]string) {
  refreshEndpoint := "https://accounts.spotify.com/api/token"
  
  values := url.Values{}
  values.Add("grant_type", "refresh_token")
  values.Add("client_id", os.Getenv("SPOTIFY_CLIENT_ID"))
  values.Add("client_secret", os.Getenv("SPOTIFY_CLIENT_SECRET"))
  values.Add("refresh_token", os.Getenv("SPOTIFY_TOKEN_REFRESH"))

  refreshData := strings.NewReader(values.Encode())

  refreshHeaders := map[string]string{
    "content-type": "application/x-www-form-urlencoded",
  }

  response := request("POST", refreshEndpoint, refreshData, refreshHeaders)

  var tokenData RefreshTokenResponse
  if err := json.Unmarshal(response, &tokenData); err != nil {
    log.Print("Error: Failed to unmarshall response: ", err)
  }

  newToken := tokenData.AccessToken
  fmt.Println("Token refreshed: ", newToken)
  
  file := filepath.Join(SPOTIFY_TOKEN_FILEPATH, SPOTIFY_TOKEN_FILENAME)

  if err := ioutil.WriteFile(file, []byte(newToken), os.ModePerm); err != nil {
    log.Print("Error: Failed to write file: ", err)
  }

  request(method, path, options, headers)
}

func main() {
	switch action {
    case "devices":
      response := request("GET", "/devices", nil, nil)
      fmt.Println(string(response))
    case "playing":
      response := request("GET", "/currently-playing", nil, nil)
      fmt.Println(string(response))
    case "previous":
      fmt.Println("Previous")
    case "next":
      fmt.Println("Next")
    case "play":
      currentlyPlayingResponse := request("GET", "/currently-playing", nil, nil)

      var currentlyPlaying CurrentlyPlayingResponse
      if err := json.Unmarshal(currentlyPlayingResponse, &currentlyPlaying); err != nil {
        log.Print("Error: Failed to unmarshal response: ", err)
      }

      if (currentlyPlaying.IsPlaying == true) {
        request("PUT", "/pause", nil, nil)
      } else {
        request("PUT", "/play", nil, nil)
      }
    case "transfer":
      fmt.Println("Transfer")
    case "volume":
      fmt.Println("Volume")
    case "reinstall":
      fmt.Println("Reinstall")
    default:
      fmt.Println("Default")
	}
}
