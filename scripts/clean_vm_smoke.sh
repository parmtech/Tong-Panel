Resolve-Path: 
Line |
   2 |  [IO.File]::ReadAllText((Resolve-Path scripts/clean_vm_smoke.sh).Path)
     |                          ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
     | Cannot find path 'C:\Users\~MaNBiR~\Documents\Codex\2026-10-28\referenced-chatgpt-conversation-this-is-an\scripts\clean_vm_smoke.sh' because it does not exist.
MethodInvocationException: 
Line |
   2 |  [IO.File]::ReadAllText((Resolve-Path scripts/clean_vm_smoke.sh).Path)
     |  ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
     | Exception calling "ReadAllText" with "1" argument(s): "The value cannot be an empty string. (Parameter 'path')"
