# Secrets — marvin-time-tracker

This repo follows the Rubio-Enterprises sops + age standard (§6.10 of standards-design.md).

## File layout

- `.sops.yaml` — recipient rules (Copier-synced; do not edit ad-hoc — use `scripts/rotate-sops-recipients.sh` from `Rubio-Enterprises/.github`).
- `secrets/example.yaml` — committed shape doc. **No real values.**
- `secrets.staging.enc.yaml` — encrypted env for `staging`. Decryptable by org admin, backup, and CI keys.
- `secrets.prod.enc.yaml` — encrypted env for `prod`. Decryptable by org admin, backup, and per-host deploy keys ONLY (CI is NEVER a prod recipient).
- `.env`, `secrets.local.*` — **gitignored**, plaintext, developer-machine only.

## Local workflow

```bash
sops secrets.staging.enc.yaml                          # create or edit
sops -d secrets.staging.enc.yaml                       # decrypt to stdout
sops exec-env secrets.staging.enc.yaml 'mise run dev'  # run with env populated
```

direnv users: `use sops secrets.staging.enc.yaml` is in `.envrc` — `direnv allow` to enable.

## Recovery

Lost laptop, lost key: see §6.10 recovery scenario in `standards-design.md`. Backup recovery key (PGP-encrypted age key on paper) is in the safe deposit box.
