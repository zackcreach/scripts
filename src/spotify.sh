#!/bin/bash

# NOTE: requires json library (install using 'npm i -g json')

ARG1=$1
ARG2=$2
ARG3=$3

SPOTIFY_TOKEN_FILEPATH="$HOME/Library/Preferences/Spotify"
SPOTIFY_TOKEN_FILENAME="spotify_access_token.txt"
SPOTIFY_TOKEN_LOCAL=$(cat "$SPOTIFY_TOKEN_FILEPATH/$SPOTIFY_TOKEN_FILENAME" || echo 'PLACEHOLDER')
VOLUME_VALUE_DEFAULT=5

request() {
  local method=$1
  local path=$2
  local options=$3
  local host='https://api.spotify.com/v1/me/player'

  local response=$(curl --request $method "$host$path" \
    --header "Authorization: Bearer $SPOTIFY_TOKEN_LOCAL" \
    $options
  )
  local error=$(echo $response | json -q error)

  if [[ $error ]]; then
    local status=$(echo $error | json -q status)

    # Handle 401 expired error
    if [[ $status == 401 ]]; then
      SPOTIFY_TOKEN_LOCAL=$(update_token)
      echo $(curl --request $method "$host$path" \
        --header "Authorization: Bearer $SPOTIFY_TOKEN_LOCAL" \
        $options
      )
    fi
  else
    echo $response
  fi
}

update_token() {
  # Request new token based on refresh_token and save response
  local response=$(curl --request POST \
    --url https://accounts.spotify.com/api/token \
    --data grant_type=refresh_token \
    --data client_id=$SPOTIFY_CLIENT_ID \
    --data client_secret=$SPOTIFY_CLIENT_SECRET \
    --data refresh_token=$SPOTIFY_TOKEN_REFRESH \
    --header 'content-type: application/x-www-form-urlencoded')

  # Extract access_token value from response JSON
  local new_token=$(echo $response | json -q access_token)

  # Write new token to .zshrc for next script execution
  mkdir -p $SPOTIFY_TOKEN_FILEPATH

  echo $new_token > "$SPOTIFY_TOKEN_FILEPATH/$SPOTIFY_TOKEN_FILENAME"

  # Return new token
  echo $new_token
}

case $ARG1 in
  'devices')
    devices=$(request 'GET' '/devices' | json -q devices)
if [[ $ARG2 ]]; then
      echo $devices | json -q -c "this.name === '$ARG2'"
    else
      echo $devices | json -q
    fi
    ;;

  'playing')
    response=$(request 'GET' '/currently-playing')
    echo $response | json -q 
    ;;


  'previous')
    response=$(request 'POST' '/previous')
    ;;

  'next')
    response=$(request 'POST' '/next')
    ;;

  'play')
    is_playing=$(request 'GET' '/currently-playing' | json -q is_playing)
    if [[ $is_playing == true ]]; then
      response=$(request 'PUT' '/pause')
    elif [[ $is_playing == false ]]; then
      response=$(request 'PUT' '/play')
    fi
    ;;

  'transfer')
    response=$(request 'GET' '/devices')
    devices=$(echo $response | json -q devices)

    if [[ $ARG2 ]]; then
      device_id=$(echo $devices | json -q -c "this.name === \"$ARG2\"" | json -q [0]id)
    else
      device_id=$(echo $devices | json -q [0]id)
    fi

    request 'PUT' '' "--header Content-Type:application/json \
      --header Accept:application/json \
      --data {\"device_ids\":[\"$device_id\"]}"
    ;;

  'volume')
    # Set direction to modify volume
    if [[ $ARG2 == 'up' || $ARG2 == 'down' || $ARG2 == 'mute' ]]; then
      direction=$ARG2
    else
      echo 'Error: Specify either \'up\' or \'down\' as second argument
    fi

    # Set custom value to increment/decrement, defaulting to 5
    if [[ $ARG3 =~ [0-9]+ ]]; then
      value=$ARG3
    else
      value=$VOLUME_VALUE_DEFAULT
    fi

    # Get devices list and then filter by active devices to find volume percent
    devices=$(request 'GET' '/devices' | json -q devices)
    volume_percent=$(echo $devices | json -q -c 'this.is_active' | json -q [0]volume_percent)

    if [[ $direction == 'up' ]]; then
      volume_percent_modified=$(($volume_percent + $value))
    elif [[ $direction == 'down' ]]; then
      volume_percent_modified=$(($volume_percent - $value))
    elif [[ $direction == 'mute' ]]; then
      volume_percent_modified=0
    fi

    volume_percent_query="?volume_percent=$volume_percent_modified"
    response=$(request 'PUT' "/volume$volume_percent_query")
    ;;
  
  'reinstall')
    src_path="$HOME/scripts/src/spotify.sh"
    bin_path="$HOME/scripts/bin/spotify"

    sudo cp $src_path $bin_path
    if [[ $? == 0 ]]; then
      echo "Successfully reinstalled at $bin_path"
    fi
    ;;
esac
