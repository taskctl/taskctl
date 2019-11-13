class Wilson < Formula
  desc "Wilson the task runner"
  homepage "https://github.com/trntv/wilson"
  url "https://github.com/trntv/wilson/releases/download/0.1.0-alpha-2/wilson_darwin"
  sha256 "1c80afdfd20919233eaf23a2f9e7edd3420ca064c73f72335761a1eb46ae12e0"

  def install
    bin.install "wilson"
  end
end
