# ${{ values.repoName }}

${{ values.description }}

This project is created from an opinionated template including CI/CD, GitOps and more from day 1


## How to deploy to development

Push some code changes on the master branch. Any commit on master will be deployed in the development environment.

## How to deploy to production

- Tag the commit you want deployed to production starting with a `v` such as :
    - v1.0.0
    - v1.0-beta1
    - vSomeVersion
- Push it
- It will be automatically deployed to production

```
git tag <your_tag> -m 'Some optional release message'
git push --tags
# enjoy
```
