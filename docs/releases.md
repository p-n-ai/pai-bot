# Releases

Every successful `CI` run for a push to `main` starts `Nightly candidate`. The app and admin ECR repositories must enforce immutable tags. The workflow builds each missing image once, tags it with the full commit SHA, resolves its immutable digest, and uploads `candidate.json` as `nightly-candidate-<sha>`. A rerun reuses the existing artifact, while an interrupted run resumes only the missing image.

To promote a candidate, run `Stable release` with the successful nightly workflow run ID and a new `vMAJOR.MINOR.PATCH` version. The workflow verifies the nightly artifact and its originating successful `main` CI run, checks out the recorded SHA, and deploys the recorded image digests through the protected `production` environment. It creates the semantic tag and GitHub Release only after deployment succeeds.

Promotion fails before deployment when the version is malformed, an existing tag points to another commit, the candidate metadata is missing or ambiguous, the candidate belongs to another repository, or its source CI run does not match the recorded SHA. Stable promotion never rebuilds application images. If tag creation succeeds but release creation fails, rerunning the same version and candidate safely completes the missing GitHub Release.

Database migrations in a stable candidate must remain backward-compatible with the previously deployed application because automated rollback restores application images, not database state. The pull request risk section must state the coordinated rollout or recovery procedure for any migration that cannot satisfy that constraint.
