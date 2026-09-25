class Netductor < Formula
  desc "netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.17"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.17/netductor-darwin-arm64"
      sha256 "6337c0a71c07757b544decf98c829fd614e690137cdfb7e2b9a95e1c9dfd2e74"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.17/netductor-darwin-amd64"
      sha256 "511fce2b8efea680a14e08ad0af6e0b0f68dd2fb0550d96f0f3cea8baeaf3bf2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.17/netductor-linux-amd64"
      sha256 "f2867f08cddc57fe4b0007cc32485fa86e3e326338a549893e0d7a7cbbcb02bd"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.17/netductor-linux-arm64"
      sha256 "34ea6c3acdf20f95be78fe1764b5a6fc817a208522330f954c7757fda6b8d823"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
end
