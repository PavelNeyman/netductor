class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.85"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.85/netductor-darwin-arm64"
      sha256 "9ec0dd5fb56945ceee6efabe725d6a9c21100d245efe6d6b6c7e1c3b54d5d0a3"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.85/netductor-darwin-amd64"
      sha256 "0cfe4654946dd2419669b7084fd1050f79d3f9e32249dd435698fb54b5252f0c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.85/netductor-linux-amd64"
      sha256 "e43f15dbaeeddd40e7373dff2ae720e3c34094df35da716ae911a7f8ea9be2be"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.85/netductor-linux-arm64"
      sha256 "f047b949e85c5c9bcaa96d17dd74e9af86ae1381109841b5f62f6a67194f9a8c"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
