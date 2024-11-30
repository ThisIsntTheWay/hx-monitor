#!/bin/bash
set -e

# -------------------------------
# Build whisper

if [ ! -f "libwhisper.a" ]; then
    echo "> Building whisper libs..."

    REPO_URL="https://github.com/ggerganov/whisper.cpp.git"
    CLONE_DIR="whisper.cpp"
    BINDINGS_GO_DIR="bindings/go"
    WHISPER_TARGET_TAG="v1.7.2"
    LAST_DIR=$(pwd)

    if [ ! -d "$CLONE_DIR" ]; then
        git clone "$REPO_URL" "$CLONE_DIR"
    fi

    cd "$CLONE_DIR"
    git checkout tags/$WHISPER_TARGET_TAG

    cd "$BINDINGS_GO_DIR"
    make whisper

    cd $LAST_DIR
    echo "> Done building whisper"
fi

# -------------------------------
# Build binary
export C_INCLUDE_PATH="$CLONE_DIR/include"
export LIBRARY_PATH="$CLONE_DIR/"

go build .