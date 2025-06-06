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
    rev = "8f6982fa77d227ba9344dc5fa32d20282198e904";
    sha256 = "sha256-002933pqvqmz6jmzz7x8iy36wxdnd92jh2gikxf249d5msdbw8mc";
  };

  vendorHash = "sha256-0zjjlfndxqclkhdwkc3z7clz42i3ir160qgw2py070qzjy37y87p";

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