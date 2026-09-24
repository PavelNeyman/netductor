class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.82"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.82/netductor-darwin-arm64"
      sha256 "c49b85f450888ffc15109675426104cb639913f7b59a993e515f524c30e9f06f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.82/netductor-darwin-amd64"
      sha256 "3210bf01ef798998461f46a3c2673e6f93440608316736df56581bdbf044bef1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.82/netductor-linux-amd64"
      sha256 "3cb2194f0f8392c35b4a7348eb112d64511268ed8c978974a345934c4a7626e6"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.82/netductor-linux-arm64"
      sha256 "ea234220acb3c0788382c73c7261a0922c83f3119ffbda74f40a8d63c8d3731f"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
