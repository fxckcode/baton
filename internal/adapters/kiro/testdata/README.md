# Kiro fixtures

The command `kiro-cli chat --no-interactive "/usage"` is documented:
https://kiro.dev/docs/reference/cli-commands/ and https://kiro.dev/docs/reference/slash-commands/.
The text layout is not in those pages.

- `overage.txt` is the non-interactive sample from
  https://github.com/aws/amazon-q-developer-cli/issues/3867 (`Credits (15312.32 of 10000 covered in plan)`).
- `bonus.txt` follows the bonus line described in
  https://github.com/steipete/CodexBar/blob/main/docs/kiro.md (`X/Y credits used`). The numbers are synthetic.
- `bar.txt` is only a meter. It must not become a percent, because the docs do not say whether a bare bar is used or remaining.

None of these were captured from a logged-in Kiro CLI on this machine.
`ansi.txt` wraps the overage sample in a color sequence.
