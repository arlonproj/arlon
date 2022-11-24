# How to contribute to Arlon
Team Arlon welcomes and encourages everyone to participate in its development via pull requests on GitHub.

We prefer to take in pull requests to our active development branch i.e. the `main` branch. To report a bug or request a feature, we rely on
GitHub issues. There are a number of points to keep in mind when submitting a feature request, reporting a bug or contributing in the development of Arlon.

1. Before making a feature request, or reporting a bug please browse through the existing open issues to be sure that it hasn't been already tracked.
1. If a feature request is subsumed by some other open issue, please add your valuable feedback as a comment to the issue.
1. If a bug discovered by you is already being tracked, please provide additional information as you see fit(steps to reproduce, particulars of the environment, version information etc.) as a comment.
1. Before submitting code for a new feature(or a complex, untracked bugfix) please create a new issue. This issue needs to undergo a review process which may involve a discussion on the same GitHub issue to discuss possible approaches and motivation for the said proposal.
1. Please reach out to us on [Slack](https://join.slack.com/t/arlon-io/shared_invite/zt-1kei6vw9e-XEYe_msqgEnKEgTq2jAdrw) for discussions, help, questions and the roadmap.

## Code changes
Open a pull request (PR) on GitHub following the typical GitHub workflow [here](https://github.com/arlonproj/arlon/pulls). Most of the new code changes
are merged to the `main` branch except backports, bookkeeping changes, library upgrades and some bugs that manifest only a particular version. Before contributing
new code, contributors are encouraged to either write unit tests, e2e tests or perform some form of manual validation as a sanity-check. Please adhere to standard
good practices for Golang and do ensure that the code is properly formatted and `vet` succeeds, for which we have `fmt` and `vet` targets respectively.

[//]: # (add link to e2e doc)
Since, Arlon is a growing project various areas require improvements- improving code coverage with unit tests, [e2e tests](./e2e_testing.md), documentation, CI/CD pipelines using
GitHub Actions are a few to name, we highly encourage to contribute to those areas to start with. The [e2e test documentation](./e2e_testing.md) is an excellent starting point
to grasp the workings of our e2e test setup.

## Issues / Bug reports
We track issues on [GitHub](https://github.com/arlonproj/arlon/issues/). You are encouraged to browse through these, add relevant feedback, create new issues or
participate in the development. If you are interested in a particular issue or feature request, please leave a comment to reach out to the team. In particular, the issues labeled
as [help wanted](https://github.com/arlonproj/arlon/labels/help%20wanted) are a great starting point for adding code changes to the project.

## Documentation

The documentation for Arlon is hosted on [Read the Docs](https://arlon.readthedocs.io/en/latest/) and comprises of
contents from the "docs" directory of Arlon source.

For making changes to the documentation, please follow the below steps:

1. Fork the Arlon repository on GitHub and make the desired changes.
1. Prerequisites
    1. Ensure that `python3`, `pip3` is installed.
    1. Optionally, create a `venv` by running `python3 -m venv ./venv` to create a virtual environment if you don't have one.
    1. From the root of the Arlon repository, run `pip3 install -r docs/requirements.txt` to install `mkdocs` and other pre-requisites.
1. To test your local changes, run `mkdocs serve` from the repository root.
    1. This starts a local server to host the documentation website where you can preview the changes.
1. To publish the changes, just push the changes to your fork repository and open a PR (pull request).
1. Once your PR is accepted by one of the maintainers/ owners of Arlon project, the Arlon website will be updated.
