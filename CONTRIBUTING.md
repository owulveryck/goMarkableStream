# Contributing to the goMarkableStream project

Hi there! We're thrilled that you'd like to contribute to the goMarkableStream project/product. 

This project is, as of today (september 2021), a toy project that I (owulveryck) maintain on my free time.
The CLI toEpub is part of my daily routine, and I am looking to make it more robust so others can use it safely.
This takes some time, and, as an open source maintener, I am greatful that you are considering helping this project to give some time and some skills to the community.

Following these guidelines helps to communicate that you respect the time of the developers managing and developing this open source project. In return, they should reciprocate that respect in addressing your issue, assessing changes, and helping you finalize your pull requests.

## Looking for support?

If you want to contribute and your are lost in the code, you can fill an issue, and I will do my very best to help you.

## How to report a bug

Think you found a bug? Please check [the list of open issues](https://github.com/owulveryck/goMarkableStream/issues) to see if your bug has already been reported. If it hasn't please [submit a new issue](https://github.com/owulveryck/goMarkableStream/issues/new).

Here are a few tips for writing *great* bug reports:

* Describe the specific problem (e.g., "widget doesn't turn clockwise" versus "getting an error")
* Include the steps to reproduce the bug, what you expected to happen, and what happened instead
* Check that you are using the latest version of the project and its dependencies
* Include what version of the project your using, as well as any relevant dependencies
* Only include one bug per issue. If you have discovered two bugs, please file two issues
* Even if you don't know how to fix the bug, including a failing test may help others track it down

**If you find a security vulnerability, do not open an issue. Please email olivier.wulveryck à gmail.com instead.**

## How to suggest a feature or enhancement

If you find yourself wishing for a feature that doesn't exist in the project, you are probably not alone. There are bound to be others out there with similar needs. Many of the features that the Minimal theme has today have been added because our users saw the need.

Feature requests are welcome. But take a moment to find out whether your idea fits with the scope and goals of the project. It's up to you to make a strong case to convince the project's developers of the merits of this feature. Please provide as much detail and context as possible, including describing the problem you're trying to solve.

[Open an issue](https://github.com/owulveryck/goMarkableStream/issues/new) which describes the feature you would like to see, why you want it, how it should work, etc.

## Your first contribution

We'd love for you to contribute to the project. Unsure where to begin contributing to the project? You can start by looking through these "good first issue" and "help wanted" issues:

* [Good first issues](https://github.com/owulveryck/goMarkableStream/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) - issues which should only require a few lines of code and a test or two
* [Help wanted issues](https://github.com/owulveryck/goMarkableStream/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) - issues which may be a bit more involved, but are specifically seeking community contributions

*p.s. Feel free to ask for help; everyone is a beginner at first* :smiley_cat:

## How to propose changes

Here's a few general guidelines for proposing changes:

* Each pull request should implement **one** feature or bug fix. If you want to add or fix more than one thing, submit more than one pull request
* Do not commit changes to files that are irrelevant to your feature or bug fix
* Don't bump the version number in your pull request (it will be bumped prior to release)
* Write [a good commit message](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)

At a high level, [the process for proposing changes](https://guides.github.com/introduction/flow/) is:

1. [Fork](https://github.com/pages-themes/minimal/fork) and clone the project
2. Make sure the tests pass on your machine
3. Create a new branch: `git checkout -b my-branch-name`
4. Make your change, add tests, and make sure the tests still pass
5. Push to your fork and [submit a pull request](https://github.com/owulveryck/goMarkableStream/compare)
6. Pat your self on the back and wait for your pull request to be reviewed and merged

**Interesting in submitting your first Pull Request?** It's easy! You can learn how from this *free* series [How to Contribute to an Open Source Project on GitHub](https://egghead.io/series/how-to-contribute-to-an-open-source-project-on-github)

## Bootstrapping your local development environment

`GO111MODULES=off go get github.com/owulveryck/goMarkableStream`

## Running tests

`go test ./...`

## Code of conduct

This project is governed by [the Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Additional Resources

* [Contributing to Open Source on GitHub](https://guides.github.com/activities/contributing-to-open-source/)
* [Using Pull Requests](https://help.github.com/articles/using-pull-requests/)
* [GitHub Help](https://help.github.com)