# How to contribute to Arlon
Team Arlon welcomes and encourages everyone to participate in its development via pull requests on GitHub. 

We prefer to take in pull requests to our active development branch i.e. the `main` branch. To report a bug or request a feature, we rely on 
GitHub issues. There are a number of points to keep in mind when submitting a feature request, reporting a bug or contributing in the development of Arlon.

1. Before making a feature request, or reporting a bug please browse through the existing open issues to be sure that it hasn't been already tracked.
2. If a feature request is subsumed by some other open issue, please add your valuable feedback as a comment to the issue.
3. If a bug discovered by you is already being tracked, please provide additional information as you see fit(steps to reproduce, particulars of the environment, version information etc.) as a comment.
4. Before submitting code for a new feature(or a complex, untracked bugfix) please create a new issue. This issue needs to undergo a review process which may involve a discussion on the same GitHub issue to discuss possible approaches and motivation for the said proposal.
5. We encourage to look at open issues as a starting point for making contributions. If you are interested in a particular issue or feature request, please leave a comment to reach out to the team.
6. Please reach out to us on [Slack]() for discussions, help, questions and the roadmap.

## Code changes 
Open a pull request (PR) on GitHub following the typical GitHub workflow [here](https://github.com/arlonproj/arlon/pulls). Most of the new code changes 
are merged to the `main` branch except backports, bookkeeping changes, library upgrades and some bugs that manifest only a particular version. Before contributing 
new code, contributors are encouraged to either write unit tests, e2e tests or perform some form of manual validation as a sanity-check. 
Since Arlon, is a nascent project, various areas require improvements- unit and e2e tests, documentation, code coverage, CI/CD pipelines using 
GitHub Actions are a few to name, we highly encourage to contribute to those areas as a first. The e2e test documentation is an excellent starting point 
to grasp the workings of our e2e test setup.

## Issues / Bug reports
We track issues on [GitHub](https://github.com/arlonproj/arlon/issues/). You are encouraged to browse through these, add relevant feedback, create new issues or 
participate in the development.

## Documentation
Fork the Arlon repository on GitHub and make the desired changes. 

To test your local changes, run `mkdocs serve`

To publish the changes, just push the changes to your fork repository and open a PR (pull request). 
Once your PR is accepted by one of the maintainers/ owners of Arlon project, the Arlon website will be updated.



