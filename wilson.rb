class Wilson < Formula
  version "0.1.0-alpha-7"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "d3e16028f3c5f7bf1f31fba7c2a3db23577fa90c03caa3728867e904fa387663"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
