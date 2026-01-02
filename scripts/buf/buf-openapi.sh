#!/usr/bin/env sh
set -eo pipefail

go tool buf dep update
go tool buf generate --template ./proto/buf.gen.openapi.yaml

cd gen

yq eval -i 'del(.tags)' openapi.yaml
yq eval -i 'del(.paths[][].tags)' openapi.yaml

yq eval '.paths | keys | .[]' openapi.yaml | while IFS= read -r path; do
  normalizedPath="$path"
  paramName=""
  paramDescription=""

  if [ "$normalizedPath" != "$path" ]; then
    oldPath="$path"
    YQ_OLD_PATH="$oldPath" YQ_NEW_PATH="$normalizedPath" yq eval -i '
      .paths[strenv(YQ_NEW_PATH)] = .paths[strenv(YQ_OLD_PATH)]
    ' openapi.yaml
    # Avoid deleting here to prevent aliasing issues; purge wildcards after loop.
    path="$normalizedPath"
  fi

  if printf '%s' "$path" | grep -q '^/ibc/'; then
    module="ibc"
  else
    module=$(printf '%s' "$path" | cut -d/ -f3)
  fi

  case "$path" in
  *)
    opName="${path##*/}"
    ;;
  esac

  for method in get post; do
    if yq eval ".paths[\"$path\"].$method" openapi.yaml | grep -qv '^null$'; then
      # sanitize opName if it contains path params like {denom}
      if printf '%s' "$opName" | grep -q '{'; then
        baseSegment=$(printf '%s' "$path" | awk -F/ '{print $(NF-1)}')
        opName="$baseSegment"
      fi

      newOpId="${module}_${opName}"

      if [ -n "$paramName" ]; then
        YQ_PATH="$path" YQ_METHOD="$method" YQ_PARAM="$paramName" YQ_DESC="$paramDescription" \
          yq eval -i '
            .paths[strenv(YQ_PATH)][env(YQ_METHOD)].parameters = (
              (.paths[strenv(YQ_PATH)][env(YQ_METHOD)].parameters // [])
              | map(select(.name != env(YQ_PARAM) or .in != "path"))
              + [{
                "name": env(YQ_PARAM),
                "in": "path",
                "required": true,
                "schema": {"type": "string"},
                "description": env(YQ_DESC)
              }]
            )
          ' openapi.yaml
      fi

      # set operationId
      yq eval -i \
        '.paths["'"$path"'"].'"$method"'.operationId = "'"$newOpId"'"' \
        openapi.yaml

      # set a human-friendly summary for display (Title Case of opName)
      prettyName=$(printf '%s' "$opName" | tr '_' ' ' | awk '{for(i=1;i<=NF;i++){ $i=toupper(substr($i,1,1)) substr($i,2)}; print}')
      YQ_PATH="$path" YQ_METHOD="$method" YQ_SUMMARY="$prettyName" yq eval -i \
        '.paths[strenv(YQ_PATH)][env(YQ_METHOD)].summary = strenv(YQ_SUMMARY)' \
        openapi.yaml

      # set the tag array; force acronyms/names to desired forms
      tagName="$module"
      if [ "$module" = "ibc" ]; then
        tagName="IBC"
      elif [ "$module" = "nft" ]; then
        tagName="NFT"
      elif [ "$module" = "cwhooks" ]; then
        tagName="Hooks"
      elif [ "$module" = "wasm" ]; then
        tagName="CosmWasm"
      fi
      yq eval -i \
        '.paths["'"$path"'"].'"$method"'.tags = ["'"$tagName"'"]' \
        openapi.yaml
    fi
  done
done

# collect all tags from the paths, remove duplicates and add them to the tags array
yq eval -i '
  .tags = (
    [ .paths.*.*.tags[] ]
    | unique
    | map({"name": ., "description": ""})
  )
' openapi.yaml

# remove any leftover wildcard (**) paths after normalization to avoid duplicates
yq eval -i '
  .paths |= with_entries(select(.key | contains("**") | not))
' openapi.yaml

# capitalize all tags
yq eval -i '
  .tags[].name |= (
    capture("(?<first>.)(?<rest>.*)")
    | .first |= upcase
    | .first + .rest
  ) |

  .paths[][].tags |= map(
    capture("(?<first>.)(?<rest>.*)")
    | .first |= upcase
    | .first + .rest
  )
' openapi.yaml

# sort all path keys alphabetically
yq eval -i '.paths = (.paths | to_entries | sort_by(.key) | from_entries)' openapi.yaml

# sort top-level tags alphabetically
yq eval -i '.tags |= sort_by(.name)' openapi.yaml

# sort components alphabetically
yq eval -i '
  .components.schemas = ((.components.schemas // {}) | to_entries | sort_by(.key) | from_entries)
' openapi.yaml

# move the final openapi.yaml to the correct location
mv openapi.yaml ../app/endpoints/openapi.yaml

cd ..
rm -rf gen
