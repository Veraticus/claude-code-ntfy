{ lib
, buildGoModule
, fetchFromGitHub
, stdenv
, makeWrapper
, src ? null
}:

buildGoModule rec {
  pname = "claude-code-ntfy";
  version = "0.1.0";

  # Allow local development by passing src
  source = if src != null then src else fetchFromGitHub {
    owner = "Veraticus";
    repo = "claude-code-ntfy";
    rev = "1fc583839898e82c0c2c0bc126727a6c4f610be3";
    sha256 = "sha256-06sni9va4vh99gwaf7z27fvklww8zdqyccp3hv1ncjnrw1szzvxm";
  };

  vendorHash = "sha256-0n07b6sqp4vl5vcxjydf2kzlfpmq38bnzwywp23azdnfp6dkcvg4";

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
  ];

  nativeBuildInputs = [ makeWrapper ];

  # Create a wrapper that hijacks the claude command
  postInstall = ''
    # Move the actual binary
    mv $out/bin/claude-code-ntfy $out/bin/.claude-code-ntfy-wrapped
    
    # Create the claude wrapper
    makeWrapper $out/bin/.claude-code-ntfy-wrapped $out/bin/claude \
      --add-flags "\$CLAUDE_ORIGINAL_PATH" \
      --add-flags '"$@"' \
      --run 'ORIGINAL_CLAUDE=""
IFS=":" read -ra PATH_ARRAY <<< "$PATH"
for dir in "''${PATH_ARRAY[@]}"; do
  if [[ -x "$dir/claude" ]] && [[ "$dir" != "'$out'/bin" ]]; then
    if [[ -f "$dir/../lib/node_modules/@anthropic-ai/claude-cli/package.json" ]] || \
       [[ -f "$dir/../../lib/node_modules/@anthropic-ai/claude-cli/package.json" ]] || \
       "$dir/claude" --version 2>&1 | grep -q "claude" ; then
      ORIGINAL_CLAUDE="$dir/claude"
      break
    fi
  fi
done

if [[ -z "$ORIGINAL_CLAUDE" ]]; then
  echo "Error: Original claude command not found in PATH" >&2
  echo "Please ensure Claude Code is installed via npm" >&2
  exit 1
fi

export CLAUDE_ORIGINAL_PATH="$ORIGINAL_CLAUDE"'
    
    # Also keep claude-code-ntfy as explicit command
    ln -s .claude-code-ntfy-wrapped $out/bin/claude-code-ntfy
  '';

  meta = with lib; {
    description = "Transparent wrapper for Claude Code with ntfy.sh notifications";
    homepage = "https://github.com/Veraticus/claude-code-ntfy";
    license = licenses.mit;
    maintainers = [ ];
    platforms = platforms.linux ++ platforms.darwin;
  };
}