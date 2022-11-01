FROM node:16-alpine

RUN apk --no-cache add git musl-dev openssh curl build-base

RUN mkdir -p -m 0700 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts

## Add the wait script to the image
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.9.0/wait /wait
RUN chmod +x /wait

# https://docs.docker.com/develop/develop-images/build_enhancements/#using-ssh-to-access-private-data-in-builds
RUN --mount=type=ssh git clone --depth=1 git@github.com:taikochain/taiko-mono.git

WORKDIR /taiko-mono/packages/protocol

RUN cat ./contracts/libs/LibConstants.sol | sed -i "s/TAIKO_COMMIT_DELAY_CONFIRMATIONS = 4/TAIKO_COMMIT_DELAY_CONFIRMATIONS = 1/g" ./contracts/libs/LibConstants.sol
RUN  npm pkg get version > /version.txt

RUN yarn && yarn compile

ENTRYPOINT ["node"]
