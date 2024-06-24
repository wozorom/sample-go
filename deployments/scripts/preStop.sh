#!/usr/bin/env bash

set -euo pipefail

readonly PRESTOP_SCRIPT_VERSION="@service.container.maven.mixin.version@"
readonly CURL_RETRY_AMOUNT=3
readonly CURL_TIMEOUT=${HTTP_TIMEOUT_SECS:-2}
readonly SLEEP_TIME=${SHUTDOWN_DELAY_SECS:-60}

main() {
  local eureka_url partial_instance_id instance_list instance_id

  eureka_url=$(sanitize_eureka_url "${EUREKA_URL}")

  partial_instance_id="${POD_NAME}:${SERVICE_NAME}"
  instance_list=$(fetch_instance_list "$eureka_url")
  instance_id=$(find_instance_id "$partial_instance_id" "$instance_list")

  export -f process_eureka_url

  update_instance_status "$eureka_url" "$instance_id"
  sleep "$SLEEP_TIME"
}

# Print a message to the standard error (stderr) stream with a prefix.
#
# The prefix "[preStop.sh]:" is added to the beginning of the message.
# This function is useful for providing a consistent format for error messages
# in the script.
#
# Globals:
#   None
# Arguments:
#   $* - The message to be printed to stderr.
# Returns:
#   None
std_err() {
  echo "[preStop.sh]: $*" >&2
}

####################################################################
# Sanitize Eureka URL by removing whitespaces and the '/' characters at the end of each Eureka URL.
#
# This function takes a Eureka URL comma-separate list string
# as input and removes any whitespace characters
# and the '/' character at the end of each Eureka URL
# that may be present in the input.
# This is achieved by using parameter expansion, which replaces
# all occurrences of whitespace characters with an empty string and
# sed stream editor to remove '/' at the end of line and replace all '/*,' to ','.
# Args:
#   url: The Eureka URL to sanitize (a string containing a URL).
#
# Outputs:
#   The sanitized Eureka URL without any whitespace characters and the '/' character at the end of each Eureka URL.
####################################################################
sanitize_eureka_url() {
  echo "${1//[[:space:]]/}" | sed 's/\/*$//' | sed -r 's/\/*,/,/g'
}

##########################################################################################
# Fetch the instance list from Eureka.
#
# This function retrieves the instance list from the Eureka server by making a series of
# HTTP GET requests to each Eureka node in the provided URL. It iterates over a list of
# Eureka nodes obtained by splitting the input URL on commas. The function sends an HTTP
# GET request to each node's /apps/<SERVICE_NAME> endpoint using curl.
#
# If the curl request succeeds and returns an instance list, the function echoes the
# instance list and returns. If no instance list is found after iterating through all
# nodes, the function echoes an error message and exits with a non-zero status.
#
# Args:
#   eureka_url: The Eureka URL to fetch the instance list from (a comma-separated list
#               of Eureka nodes' URLs).
#
# Outputs:
#   The instance list in JSON format.
##########################################################################################
fetch_instance_list() {
  local instance_list

  for node in $(echo "$1" | tr ',' ' '); do
    instance_list=$(curl --silent --retry $CURL_RETRY_AMOUNT --connect-timeout "$CURL_TIMEOUT" --max-time "$CURL_TIMEOUT" \
      --header "accept: application/json" \
      --header "Cache-Control: no-cache" \
      --header "User-Agent: EurekaPreStop/$PRESTOP_SCRIPT_VERSION" \
      --request GET "$node/apps/$SERVICE_NAME")

   if [[ -n "$instance_list" ]] && echo "$instance_list" | grep -q "instanceId"; then
     sanitize_eureka_url "$instance_list"
     return
   fi
  done

  std_err "Error: Instance list not found"
  exit 1
}

################################################################################
# Find the full instance ID using the partial instance ID.
#
# This function searches for the full instance ID in the provided instance list
# using a unique partial instance ID, which is POD_NAME:SERVICE_NAME.
#
# Args:
#   partial_instance_id: The unique substring of the instance ID.
#   instance_list: The list of instances to search for the full instance ID.
#
# Outputs:
#   The full instance ID corresponding to the given partial instance ID.
#
# Exits:
#   Exits with code 1 if the instance ID is not found.
################################################################################
find_instance_id() {
  local partial_instance_id instance_list instance_id

  partial_instance_id="$1"
  instance_list="$2"

  # Use grep to find the line containing the instanceId with the given partial_instance_id,
  # and then use sed to remove the instanceId label, leaving only the value.
  instance_id=$(echo "$instance_list" | grep -oE "\"instanceId\":\"${partial_instance_id}[^\"]+" | sed "s/\"instanceId\":\"//")

  # If the instance ID is not found, print an error message and exit with code 1.
  if [[ -z "$instance_id" ]]; then
    std_err "Error: Instance not found"
    exit 1
  fi

  # Output the full instance ID.
  echo "$instance_id"
}

##########################################################################################
# Sends PUT and GET requests to the Eureka server to set the instance status to
# OUT_OF_SERVICE and retrieve the instance details, respectively.
#
# This function sends a PUT request to the Eureka server to set the instance status to
# OUT_OF_SERVICE. If the request is successful, it sends a GET request to retrieve the
# instance details. If either request fails, it outputs "_FAILED_".
#
# Args:
#   url: The Eureka node URL.
#   service_name: The name of the service to deregister.
#   instance_id: The ID of the instance to deregister.
#   curl_retry_amount: The number of times to retry a curl request upon failure.
#   curl_timeout: The timeout duration (in seconds) for curl requests.
#
# Outputs:
#   If the PUT and GET requests are successful, outputs the instance details in JSON
#   format. If either request fails, outputs "_FAILED_".
##########################################################################################
process_eureka_url() {
  local url="$1"
  local service_name="$2"
  local instance_id="$3"
  local curl_retry_amount="$4"
  local curl_timeout="$5"

  curl --silent --fail --show-error --retry "$curl_retry_amount" --connect-timeout "$curl_timeout" --max-time "$curl_timeout" \
    --header "Cache-Control: no-cache" \
    --header "User-Agent: EurekaPreStop/$PRESTOP_SCRIPT_VERSION" \
    --request PUT "$url/apps/$service_name/$instance_id/status?value=OUT_OF_SERVICE"

  if [[ $? -ne 0 ]]; then
    echo -e '_FAILED_'
  else
    response=$(curl --silent --fail --show-error --retry "$curl_retry_amount" --connect-timeout "$curl_timeout" --max-time "$curl_timeout" \
      --header "accept: application/json" \
      --header "Cache-Control: no-cache" \
      --header "User-Agent: EurekaPreStop/$PRESTOP_SCRIPT_VERSION" \
      --request GET "$url/apps/$service_name/$instance_id")
    # Output the response together with a newline character
    echo -e "$response"
  fi
}

################################################################################################
# Updates the instance status to OUT_OF_SERVICE in Eureka.
#
# This function sends PUT requests in parallel to a list of Eureka nodes to set the status
# of the current pod to OUT_OF_SERVICE. It first replaces commas with newlines in the Eureka URL
# list to process each Eureka node URL individually. Then, it uses xargs with -P 0 to send
# requests in parallel to all Eureka nodes.
#
# For each response received, the function checks if the instance status has been updated
# to OUT_OF_SERVICE. If not, it prints an error message and exits with a non-zero status.
# If the status update is successful, the function outputs a success message with the updated
# instance ID.
#
# Globals:
#   CURL_RETRY_AMOUNT: The number of times to retry a failed curl request.
#   CURL_TIMEOUT: The maximum amount of time for the curl request to connect and execute.
#   SERVICE_NAME: The name of the service to update the status for.
#
# Args:
#   eureka_url: A comma-separated list of Eureka nodes' URLs.
#   instance_id: The instance ID of the pod to update the status for.
#
# Outputs:
#   A success message with the updated instanceId if the update is successful.
#   An error message if the instance status update fails.
################################################################################################
update_instance_status() {
  local eureka_url instance_id

  eureka_url="$1"
  instance_id="$2"

  local total_nodes non_responsive_nodes non_out_of_service_nodes
  total_nodes=$(echo "$eureka_url" | tr ',' ' ' | wc -w)
  non_responsive_nodes=0
  non_out_of_service_nodes=0

  output=$(echo "$eureka_url" |
    tr ',' '\n' |
    xargs -I {} -P 0 bash -c "process_eureka_url {} \"$SERVICE_NAME\" \"$instance_id\" \"$CURL_RETRY_AMOUNT\" \"$CURL_TIMEOUT\"")

   while read -r response; do
     if [[ -z "$response" ]]; then
       continue
     fi
    if [[ "$response" == "_FAILED_" ]]; then
      non_responsive_nodes=$((non_responsive_nodes + 1))
    else
      sanitized_response=$(echo "$response" | tr -d ' ')
      status=$(echo "$sanitized_response" | grep -oP '"status":"\K[^"]+')
      if [[ "$status" != "OUT_OF_SERVICE" ]]; then
        non_out_of_service_nodes=$((non_out_of_service_nodes + 1))
      fi
    fi
  done < <(echo "$output")

  if [[ $non_responsive_nodes -eq $total_nodes ]]; then
    std_err "Error: All Eureka nodes are irresponsive"
    exit 1
  elif [[ $non_out_of_service_nodes -gt 0 ]]; then
    std_err "Error: Some Eureka nodes responded with a non-OUT_OF_SERVICE status"
    exit 1
  else
    echo "removing node out of load-balancer, instanceId: $instance_id"
  fi
}

main "$@"
