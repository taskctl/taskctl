class Wilson < Formula
  version "0.1.0"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "635034cc0c389e8a169e9752835eb8fe0a2e3304cde0d1904178712db1716e2e"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
