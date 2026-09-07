# Regenerates docs/images/demo.gif. Run from the repo root:
#   vhs docs/demo/demo.tape
# Builds ai-dash and points it at a throwaway fake $HOME populated with
# synthetic sessions (docs/demo/gen_demo_data.py) so no real data is used.

Output docs/images/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Padding 20
Set WindowBar Colorful
Set TypingSpeed 60ms
Set Theme "nord"

Hide
Type 'export DEMO_ROOT=$(mktemp -d)'
Enter
Type 'python3 docs/demo/gen_demo_data.py "$DEMO_ROOT/home"'
Enter
Type 'go build -o "$DEMO_ROOT/ai-dash-demo" ./cmd/ai-dash'
Enter
Type 'export HOME="$DEMO_ROOT/home"'
Enter
Type 'export XDG_CONFIG_HOME="$HOME/.config"'
Enter
Type 'export XDG_DATA_HOME="$HOME/.local/share"'
Enter
Type 'export PATH="$DEMO_ROOT:$PATH"'
Enter
Type "clear"
Enter
Show

Type "ai-dash-demo"
Enter
Sleep 2s

Down 3
Sleep 800ms

Type "/"
Sleep 400ms
Type "invoice"
Sleep 1s
Escape
Sleep 500ms

Type "t"
Sleep 700ms
Enter
Sleep 400ms
Down
Sleep 300ms
Enter
Sleep 900ms
Escape
Sleep 700ms

Type "!"
Sleep 1200ms
Type "!"
Sleep 500ms

Type "?"
Sleep 1500ms
Type "?"
Sleep 500ms

Type "q"
Sleep 500ms
