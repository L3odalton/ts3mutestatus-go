# TS3MuteStatus

A Go application that synchronizes TeamSpeak 3 mute status with Home Assistant input_boolean entities.

## Description

This application monitors your TeamSpeak 3 client's mute status (both input and output) and synchronizes it with a Home Assistant input_boolean entity. This allows you to create automations based on your TeamSpeak mute status.

The input_boolean will be:
- ON when both input and output are unmuted
- OFF when either input or output is muted

## Prerequisites

- TeamSpeak 3 Server with Query access
- Home Assistant instance
- Docker (optional)
- Go 1.24 or later (if building from source)

## Configuration

The application requires the following environment variables:

TS3_API_KEY=your_ts3_api_key
TS3_ADDRESS=your_ts3_server:25639
HA_BASE_URL=http://your_homeassistant:8123
HA_TOKEN=your_long_lived_access_token
HA_ENTITY_ID=input_boolean.teamspeak_mic_status

### Environment Variables Explanation

- `TS3_API_KEY`: Your TeamSpeak 3 server query API key
- `TS3_ADDRESS`: Your TeamSpeak 3 server address and query port
- `HA_BASE_URL`: Your Home Assistant instance URL with port
- `HA_TOKEN`: A long-lived access token from Home Assistant
- `HA_ENTITY_ID`: The input_boolean entity ID to sync with

## Installation

### Using Docker

1. Create a `.env.local` file with your configuration (see Configuration section)

2. Pull and run the container:

docker run -d \
  --env-file .env.local \
  linusbaumann/ts3mutestatus-go:latest

### Building the Docker Image

If you want to build the Docker image yourself:

# Build the image
docker build -t ts3mutestatus-go .

# Run the container
docker run -d \
  --env-file .env.local \
  ts3mutestatus-go

### Building from source

# Initialize Go module
go mod init ts3mutestatus-go
go mod tidy

# Build the binary
go build

# Run the application
./ts3mutestatus-go

## Home Assistant Setup

1. Create an input_boolean entity in your Home Assistant configuration:

input_boolean:
  teamspeak_mic_status:
    name: TeamSpeak Mic Status
    icon: mdi:microphone

2. Create a long-lived access token in Home Assistant:
   - Profile > Long-Lived Access Tokens > Create Token

## License

MIT License

## Author

Linus Baumann <keen.key5715@linus-baumann.de>

## Contributing

Feel free to open issues or submit pull requests if you have suggestions for improvements.