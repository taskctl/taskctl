class Wilson < Formula
  version "0.1.0-beta"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "d3286463d7d30e1a94ea732d5da7552e8a9344a41d7606de9dbc4048f30fab10"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
