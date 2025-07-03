#!/bin/bash

# --- Script Function Description ---
# This script is used to clone a specific reference (tag or branch) and its corresponding commit history
# from a Git repository to a local machine, and then push it to a new Git repository.
# The script prioritizes ensuring the commit objects are uploaded.
# If the target 'main' branch is empty, it will be initialized.
# If 'main' is not empty, commit objects will be pushed to a temporary branch first.
# It supports using different authentication tokens for the source and destination repositories.

# --- 1. Parameter Initialization and Default Settings ---
# Required parameters
SOURCE_REPO_URL=""
DEST_REPO_URL=""
SOURCE_REF="" # The name of the reference to clone (e.g., main, v1.0.0)

# Optional parameters, with default values or obtained from environment variables
LOCAL_CLONE_DIR="temp_clone_dir_$(date +%Y%m%d%H%M%S)" # Name of the local temporary clone directory, with timestamp

# Source repository authentication information
# Prioritize reading from environment variables, if not set, try to use hardcoded defaults (please replace!)
SOURCE_GIT_USERNAME="${SOURCE_GIT_USERNAME:-oauth2}"
SOURCE_GIT_TOKEN="${SOURCE_GIT_TOKEN:-YOUR_SOURCE_REPO_DEFAULT_TOKEN_HERE}" # PAT for the source repository

# Destination repository authentication information
DEST_GIT_USERNAME="${DEST_GIT_USERNAME:-oauth2}"
DEST_GIT_TOKEN="${DEST_GIT_TOKEN:-YOUR_DEST_REPO_DEFAULT_TOKEN_HERE}" # PAT for the destination repository

# --- Help Function ---
show_help() {
  echo "Usage: $0 [OPTIONS]"
  echo "Clones a specified reference (branch or tag) from a source Git repository and pushes it to a target repository."
  echo ""
  echo "Options:"
  echo "  --from-repo-url <URL>        URL of the source repository (e.g., https://gitlab.com/user/repo.git)"
  echo "  --to-repo-url <URL>          URL of the target repository (e.g., https://gitlab.com/user/new-repo.git)"
  echo "  --from-ref <REF_NAME>        Name of the reference to clone (e.g., main, v1.0.0)"
  echo "  --clone-dir <DIRECTORY>      Name of the local temporary clone directory (Optional, defaults to a timestamped temp dir)"
  echo ""
  echo "  --source-username <USERNAME> Username for cloning the source repository (Optional, defaults to oauth2 or env var SOURCE_GIT_USERNAME)"
  echo "  --source-token <TOKEN>       PAT for cloning the source repository (Optional, defaults to env var SOURCE_GIT_TOKEN or hardcoded default)"
  echo ""
  echo "  --dest-username <USERNAME>   Username for pushing to the target repository (Optional, defaults to oauth2 or env var DEST_GIT_USERNAME)"
  echo "  --dest-token <TOKEN>         PAT for pushing to the target repository (Optional, defaults to env var DEST_GIT_TOKEN or hardcoded default)"
  echo ""
  echo "  -h, --help                   Display this help message and exit"
  echo ""
  echo "Example:"
  echo "  $0 \\"
  echo "     --from-repo-url \"https://aml-gitlab.alaudatech.net/fy-dev/amlmodels/iris\" \\"
  echo "     --to-repo-url \"https://aml-gitlab.alaudatech.net/fy-prod/amlmodels/iris3\" \\"
  echo "     --from-ref \"v0.0.1\" \\"
  echo "     --source-token \"glpat-source-token-xxx\" \\"
  echo "     --dest-token \"glpat-dest-token-yyy\""
  echo ""
  echo "Note: It's recommended to set tokens via environment variables, e.g.:"
  echo "  export SOURCE_GIT_TOKEN=\"glpat-source-token\""
  echo "  export DEST_GIT_TOKEN=\"glpat-dest-token\""
}

# --- Parse Command Line Arguments ---
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --from-repo-url)
      SOURCE_REPO_URL="$2"
      shift
      ;;
    --to-repo-url)
      DEST_REPO_URL="$2"
      shift
      ;;
    --from-ref)
      SOURCE_REF="$2"
      shift
      ;;
    --clone-dir)
      LOCAL_CLONE_DIR="$2"
      shift
      ;;
    --source-username)
      SOURCE_GIT_USERNAME="$2"
      shift
      ;;
    --source-token)
      SOURCE_GIT_TOKEN="$2"
      shift
      ;;
    --dest-username)
      DEST_GIT_USERNAME="$2"
      shift
      ;;
    --dest-token)
      DEST_GIT_TOKEN="$2"
      shift
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Error: Unknown parameter '$1'"
      show_help
      exit 1
      ;;
  esac
  shift # Move to the next parameter
done

# --- 2. Parameter Validation ---
if [ -z "$SOURCE_REPO_URL" ] || [ -z "$DEST_REPO_URL" ] || [ -z "$SOURCE_REF" ]; then
  echo "Error: Missing required parameters (--from-repo-url, --to-repo-url, --from-ref)."
  show_help
  exit 1
fi

if [ -z "$SOURCE_GIT_TOKEN" ] && [[ "$SOURCE_REPO_URL" == *"github.com"* || "$SOURCE_REPO_URL" == *"gitlab.com"* || "$SOURCE_REPO_URL" == *"gitlab.alaudatech.net"* ]]; then
  echo "Error: Source repository requires authentication, but SOURCE_GIT_TOKEN is not set. Please provide it via --source-token or environment variable SOURCE_GIT_TOKEN."
  exit 1
fi

if [ -z "$DEST_GIT_TOKEN" ] && [[ "$DEST_REPO_URL" == *"github.com"* || "$DEST_REPO_URL" == *"gitlab.com"* || "$DEST_REPO_URL" == *"gitlab.alaudatech.net"* ]]; then
  echo "Error: Destination repository requires authentication, but DEST_GIT_TOKEN is not set. Please provide it via --dest-token or environment variable DEST_GIT_TOKEN."
  exit 1
fi

# Construct authenticated URLs
# URL used for cloning the source repository
SOURCE_REPO_URL_AUTH="${SOURCE_REPO_URL}"
if [ -n "$SOURCE_GIT_TOKEN" ]; then
    SOURCE_REPO_URL_AUTH="https://${SOURCE_GIT_USERNAME}:${SOURCE_GIT_TOKEN}@$(echo ${SOURCE_REPO_URL} | sed -e 's|https://||')"
fi

# URL used for pushing to the target repository
DEST_REPO_URL_AUTH="${DEST_REPO_URL}"
if [ -n "$DEST_GIT_TOKEN" ]; then
    DEST_REPO_URL_AUTH="https://${DEST_GIT_USERNAME}:${DEST_GIT_TOKEN}@$(echo ${DEST_REPO_URL} | sed -e 's|https://||')"
fi


# --- 3. Robustness Enhancement: Pre-checks and Cleanup ---
# Check if the local clone directory already exists and is not empty, to prevent accidental overwrites
if [ -d "${LOCAL_CLONE_DIR}" ]; then
  echo "Warning: Local clone directory '${LOCAL_CLONE_DIR}' already exists. Attempting to remove and recreate to avoid conflicts."
  rm -rf "${LOCAL_CLONE_DIR}"
  if [ $? -ne 0 ]; then
    echo "Error: Could not remove existing directory '${LOCAL_CLONE_DIR}'. Please remove it manually or specify a different directory name."
    exit 1
  fi
fi

# --- 4. Clone Source Repository (Full Clone to ensure all objects are present) ---
echo "--- Cloning source repository: ${SOURCE_REPO_URL} to ${LOCAL_CLONE_DIR} ---"
# Use the authenticated URL for cloning
git clone "${SOURCE_REPO_URL_AUTH}" "${LOCAL_CLONE_DIR}"
if [ $? -ne 0 ]; then
  echo "Error: Cloning source repository failed! Please check source URL, authentication details, or network connection."
  exit 1
fi

# Enter the cloned directory
cd "${LOCAL_CLONE_DIR}" || { echo "Error: Could not enter directory '${LOCAL_CLONE_DIR}'."; exit 1; }

# --- 5. Get Commit ID corresponding to Source Reference (Tag/Branch) ---
echo "--- Resolving Commit ID for reference '${SOURCE_REF}' ---"
# Use git rev-parse to ensure the exact Commit ID is obtained
# Check if it's a valid reference (tag/branch)
REF_TYPE=""
if git show-ref --verify "refs/tags/${SOURCE_REF}" &>/dev/null; then
    REF_TYPE="tag"
    COMMIT_ID_TO_PUSH=$(git rev-parse "refs/tags/${SOURCE_REF}")
elif git show-ref --verify "refs/heads/${SOURCE_REF}" &>/dev/null; then
    REF_TYPE="branch"
    COMMIT_ID_TO_PUSH=$(git rev-parse "refs/heads/${SOURCE_REF}")
else
    echo "Error: Reference '${SOURCE_REF}' not found as a tag or branch in the local repository."
    exit 1
fi

if [ $? -ne 0 ]; then
  echo "Error: Could not resolve reference '${SOURCE_REF}'! Please check if the reference exists in the source repository."
  exit 1
fi
echo "Reference '${SOURCE_REF}' is a ${REF_TYPE}, and its corresponding Commit ID is: ${COMMIT_ID_TO_PUSH}"

# --- 6. (Important) Create a local temporary branch pointing to the Commit ID ---
# This step is to create a local branch that we can push to the remote.
TEMP_LOCAL_BRANCH_NAME="temp-mirror-local-$(date +%s)" # Use timestamp for uniqueness
echo "--- Creating local temporary branch: '${TEMP_LOCAL_BRANCH_NAME}' pointing to ${COMMIT_ID_TO_PUSH} ---"
git branch "${TEMP_LOCAL_BRANCH_NAME}" "${COMMIT_ID_TO_PUSH}"
if [ $? -ne 0 ]; then
  echo "Error: Failed to create local temporary branch!"
  exit 1
fi

# --- 7. Add New Remote Repository ---
echo "--- Adding target remote repository: ${DEST_REPO_URL} as 'destination' ---"
# Use the authenticated URL
git remote add destination "${DEST_REPO_URL_AUTH}" 2>/dev/null || \
git remote set-url destination "${DEST_REPO_URL_AUTH}"
if [ $? -ne 0 ]; then
  echo "Error: Failed to add/set remote repository! Please check URL and authentication details."
  exit 1
fi

# --- 8. (Critical) Push Commit Objects to ensure they exist on the remote ---
TARGET_MAIN_BRANCH="main"
REMOTE_TEMP_BRANCH_FOR_OBJECTS="refs/heads/mirror-objects-${TEMP_LOCAL_BRANCH_NAME}"

# Check if the target 'main' branch exists on the remote
echo "--- Checking if remote '${TARGET_MAIN_BRANCH}' branch exists in '${DEST_REPO_URL}' ---"
# Use git ls-remote to check for remote branches reliably
if git ls-remote "${DEST_REPO_URL_AUTH}" "refs/heads/${TARGET_MAIN_BRANCH}" | grep -q "${TARGET_MAIN_BRANCH}"; then
    echo "Remote '${TARGET_MAIN_BRANCH}' branch exists. Pushing commit objects to a temporary remote branch."
    # If main exists, push to a temporary remote branch to upload objects
    git push destination "${TEMP_LOCAL_BRANCH_NAME}:${REMOTE_TEMP_BRANCH_FOR_OBJECTS}"
    PUSH_OBJECT_STATUS=$?
    if [ $PUSH_OBJECT_STATUS -ne 0 ]; then
        echo "Error: Failed to push commit objects to remote temporary branch! Exit code: ${PUSH_OBJECT_STATUS}."
        echo "This could be a permission issue, or the target repository rejected the commit. Please check Git error messages."
        exit 1
    else
        echo "Commit objects for '${SOURCE_REF}' successfully pushed to remote temporary branch '${REMOTE_TEMP_BRANCH_FOR_OBJECTS}'."
    fi
else
    echo "Remote '${TARGET_MAIN_BRANCH}' branch does not exist (or is empty). Attempting to initialize it with commit objects."
    # If main doesn't exist, try to initialize it
    git push destination "${TEMP_LOCAL_BRANCH_NAME}":"refs/heads/${TARGET_MAIN_BRANCH}"
    PUSH_MAIN_INIT_STATUS=$?
    if [ $PUSH_MAIN_INIT_STATUS -ne 0 ]; then
        echo "Error: Failed to initialize remote '${TARGET_MAIN_BRANCH}' branch! Exit code: ${PUSH_MAIN_INIT_STATUS}."
        echo "This might be due to permissions or repository rules. Please check Git error messages."
        exit 1
    else
        echo "Remote '${TARGET_MAIN_BRANCH}' branch successfully initialized with commit objects for '${SOURCE_REF}'."
    fi
    # Even if main was just initialized, the objects are there, so no need for the separate temporary branch push.
    # We will just manage the cleanup for REMOTE_TEMP_BRANCH_FOR_OBJECTS later if it wasn't pushed.
    REMOTE_TEMP_BRANCH_FOR_OBJECTS="" # Mark as not pushed to main.
fi


# --- 9. Push the Original Tag or Branch ---
# At this point, we have ensured all necessary commit objects are in the remote repository.
if [ "${REF_TYPE}" == "tag" ]; then
    echo "--- Second push: Pushing Tag '${SOURCE_REF}' to the target repository ---"
    git push destination "refs/tags/${SOURCE_REF}"
elif [ "${REF_TYPE}" == "branch" ]; then
    echo "--- Second push: Pushing branch '${SOURCE_REF}' to the target repository ---"
    # If the source reference is a branch, push it to a branch with the same name in the target repository
    git push destination "refs/heads/${SOURCE_REF}"
fi

if [ $? -ne 0 ]; then
  echo "Error: Failed to push reference '${SOURCE_REF}' to the target repository!"
  exit 1
fi
echo "Reference '${SOURCE_REF}' pushed successfully to the target repository."

# --- 10. (Optional) Clean up local and remote temporary branches ---
echo "--- Cleaning up local temporary branch '${TEMP_LOCAL_BRANCH_NAME}' ---"
git branch -d "${TEMP_LOCAL_BRANCH_NAME}" &>/dev/null # Hide output
if [ $? -ne 0 ]; then
  echo "Warning: Failed to delete local temporary branch. Please clean it up manually: 'cd ${LOCAL_CLONE_DIR} && git branch -d ${TEMP_LOCAL_BRANCH_NAME}'"
fi

# Clean up remote temporary branch only if it was actually pushed
if [ -n "${REMOTE_TEMP_BRANCH_FOR_OBJECTS}" ]; then
    echo "--- Cleaning up remote temporary branch '${REMOTE_TEMP_BRANCH_FOR_OBJECTS}' ---"
    git push destination --delete "${REMOTE_TEMP_BRANCH_FOR_OBJECTS}" &>/dev/null || true # Allow failure, e.g., no permissions or already deleted
    if [ $? -ne 0 ]; then
      echo "Warning: Failed to delete remote temporary branch. Please clean it up manually: 'git push ${DEST_REPO_URL} --delete ${REMOTE_TEMP_BRANCH_FOR_OBJECTS}'"
    fi
else
    echo "No remote temporary branch was created for object upload (main branch was initialized instead)."
fi


# Exit the clone directory so it can be deleted
cd ..

echo "--- Cleaning up temporary clone directory '${LOCAL_CLONE_DIR}' ---"
rm -rf "${LOCAL_CLONE_DIR}"
if [ $? -ne 0 ]; then
  echo "Warning: Failed to delete temporary clone directory. Please clean it up manually: 'rm -rf ${LOCAL_CLONE_DIR}'"
fi

echo "--- Operation Completed ---"
echo "Please check your target repository: ${DEST_REPO_URL}, to confirm that '${TARGET_MAIN_BRANCH}' branch (if initialized) and '${SOURCE_REF}' (tag/branch) have been created."
echo "You may need to change the default branch in your GitLab/GitHub repository settings."