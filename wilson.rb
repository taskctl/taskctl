class Wilson < Formula
  version "0.1.0-beta.3"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "8c1a8ee623cd354d2df0f5750398372407dcf1ba1ca545c3b9a5d5d8a546cb9a"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
