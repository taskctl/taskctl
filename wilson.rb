class Wilson < Formula
  version "0.1.0"
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "69164670eedb95f6f13cc51f60b0ff6256594401217d73f168dd1aa47f859063"

  def install
    bin.install "bin/wilson_darwin_amd64"
  end
end
