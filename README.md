# The Bus - Honolulu Transit Service
SMS/Call To get next bus arrival for the Honolulu Transit Service using the Twilio platform

# Prerequisites
- [Sign up for a Twilio Account (Free)](https://www.twilio.com/sign-up/try-twilio)
- [Sign up for an API Key with The Bus (Free)](http://api.thebus.org/)
- [Install Golang on your machine](http://golang.org/)
- [Install PostgreSQL (Optional)](http://www.postgresql.org/)

# Build Instructions
- Open your terminal and check out this repository
- cd into the directory that you cloned this into
- Execute **go run main.go db.go -key "API_KEY"** in your terminal to start it up

# Optional Run Parameter
You can enable database logging by adding **-useDB true** after the **-key "API_KEY"** argument
