#!/bin/bash

# Define the number of concurrent requests and the data array
concurrency=10
data=(
  '{"username": "test41", "email": "test41@test.com", "password": "1234"}'
  '{"username": "test42", "email": "test42@test.com", "password": "5678"}'
  '{"username": "test43", "email": "test43@test.com", "password": "abcd"}'
  '{"username": "test44", "email": "test44@test.com", "password": "abcd"}'
  '{"username": "test45", "email": "test45@test.com", "password": "abcd"}'
  '{"username": "test46", "email": "test46@test.com", "password": "abcd"}'
  '{"username": "test47", "email": "test47@test.com", "password": "abcd"}'
  '{"username": "test48", "email": "test48@test.com", "password": "abcd"}'
  '{"username": "test49", "email": "test49@test.com", "password": "abcd"}'
  '{"username": "test50", "email": "test50@test.com", "password": "abcd"}'
  # Add more dynamic data elements as needed
)

# Function to make concurrent requests
make_request() {
curl -s -X POST 'http://localhost:5002/auth/register' \
    --header 'Content-Type: application/json' \
    --data-raw "$1" \
    -o response_$2.txt
}

# Loop through the data array and make concurrent requests
sleep 2
for ((i=0; i<${#data[@]}; i++)); do
  make_request "${data[$i]}" $i &
  if (( (i+1) % concurrency == 0 )); then
    wait
  fi
done
wait
