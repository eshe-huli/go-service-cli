# CI is where the local convention becomes a merge requirement

The source repository includes CI for the CLI itself. That workflow is provided but
was not run on a remote GitHub repository during this build.

For each generated service, the required job must first install gsvc from a trusted,
pinned source commit or release artifact and verify that it matches the service's
`tool_version`. Then execute, from that service's root:

    gsvc check --verify --strict --json

Use `--race` as well on runners with the supported race toolchain. Build/test commands
execute project code, so run untrusted pull requests in appropriately restricted CI.

No public installation URL is hard-coded because this project has not been published.
Do not substitute a made-up go-install path or a floating third-party package with a
similar name. Once the owner chooses a repository, add the actual pinned distribution
step and make this check required for merges.

Require separate review for CLI source, manifests, ownership metadata, agent
instructions and CI configuration. A feature agent should not be able to redefine
its own passing conditions. Local instructions or a voluntarily executed check alone
do not enforce a team policy.
