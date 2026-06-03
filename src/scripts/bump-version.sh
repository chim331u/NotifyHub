#!/usr/bin/env bash

# Ottieni la directory dello script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
VERSION_FILE="$DIR/../VERSION"

if [ ! -f "$VERSION_FILE" ]; then
    echo "Errore: File VERSION non trovato in $VERSION_FILE"
    exit 1
fi

CURRENT_VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
IFS='.' read -r major minor patch <<< "$CURRENT_VERSION"

MODE=$1

if [ "$MODE" == "minor" ]; then
    minor=$((minor + 1))
    patch=0
elif [ "$MODE" == "major" ]; then
    major=$((major + 1))
    minor=0
    patch=0
else
    echo "Uso: $0 [minor|major]"
    exit 1
fi

NEW_VERSION="$major.$minor.$patch"
echo "$NEW_VERSION" > "$VERSION_FILE"
echo "Versione aggiornata con successo a $NEW_VERSION"
