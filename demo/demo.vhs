# https://github.com/charmbracelet/vhs/

Output demo.webp

Set FontSize 26
Set Padding 12
Set Width 1660
Set Height 80
Set Theme "Builtin Light"

Type "ec2ssh -l ec2-user i-0e2d55b1a6328d2ea  # connect using generated ephemeral SSH key"
Sleep 2s

Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Type "--use-eice ip-10-0-0-147  # tunnel through EC2 Instance Connect Endpoint"
Sleep 3s

Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Ctrl+W
Type "--profile dev --region us-east-1 app01  # use multiple accounts and regions"
Sleep 4s