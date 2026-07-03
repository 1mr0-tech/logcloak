# Governance

This document describes how the logcloak project is governed.

## Project roles

### Maintainer

Maintainers have write access to the repository and are responsible for:

- Reviewing and merging pull requests
- Triaging issues
- Cutting releases
- Setting the technical direction of the project
- Enforcing the [Code of Conduct](CODE_OF_CONDUCT.md)

Current maintainers are listed in [OWNERS](OWNERS).

### Contributor

Anyone who opens an issue, submits a pull request, or participates in project discussions is a contributor. Contributors do not have write access to the repository.

## Decision making

For routine changes (bug fixes, documentation, minor features), a single maintainer approval is sufficient to merge a pull request.

For significant changes (new features, breaking changes, architectural decisions, dependency additions), at least one additional maintainer must review and approve, or a 72-hour comment period must pass with no objections from other maintainers.

Decisions are made by consensus among maintainers. If consensus cannot be reached, the project lead (see OWNERS) has the deciding vote.

## Becoming a maintainer

A contributor may be nominated as a maintainer by an existing maintainer if they have:

- Made sustained, high-quality contributions over at least three months
- Demonstrated understanding of the project's goals and codebase
- Shown good judgment in issue triage and code review

Nominations are approved by consensus of existing maintainers via a pull request to OWNERS.

## Removing a maintainer

A maintainer may step down voluntarily by submitting a pull request to remove themselves from OWNERS. Maintainers who are unresponsive for six months may be moved to emeritus status by a majority vote of active maintainers.

## Releases

Releases follow [Semantic Versioning](https://semver.org). Any maintainer may cut a patch release. Minor and major releases require consensus among maintainers. The release process is documented in [CONTRIBUTING.md](CONTRIBUTING.md).

## Amendments

This governance document may be amended by a pull request approved by a majority of maintainers, with a minimum 72-hour review period.
