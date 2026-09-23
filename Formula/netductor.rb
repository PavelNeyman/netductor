class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.66"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.66/netductor-darwin-arm64"
      sha256 "420119e80b542e82ac342b8aa24dcf9441c0d796840c37220e9b2523f1895823"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.66/netductor-darwin-amd64"
      sha256 "8f8ee0c7a94039cd4d9f2874931e6092492582dcbd303622ebc01c4b3d535507"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.66/netductor-linux-amd64"
      sha256 "f013402ceb1cbcb5b7c5b3995ba21597649df09a8d2b47d5951d155056056e9b"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.66/netductor-linux-arm64"
      sha256 "5bf0b2c7201d03f2774c0ea025749de56595d0c70982d7c36687a775bb9886ff"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
