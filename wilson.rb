class Wilson < Formula
  version "0.1.1"
  desc "Wilson - routine tasks automation toolkit"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/#{version}/wilson-darwin-amd64.tar.gz"
  sha256 "24defdc48268be9931779a5d15f59f20776725a205a32ec09bd0ed7a27ec542a"

  def install
    bin.install "wilson_darwin_amd64"
  end
end
