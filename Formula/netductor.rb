class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.88"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.88/netductor-darwin-arm64"
      sha256 "c249bf9ec792da6671441a1e2009504d7df2a214fe6c06653697342d872dc7af"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.88/netductor-darwin-amd64"
      sha256 "904a445bc72291380c683170423c94efa981e191db94750dd29d71013604643b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.88/netductor-linux-amd64"
      sha256 "b2a15197848de79cffb9fb201a14b1de6c138d0a2f13aa13bdf7983a1a34b8f9"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.88/netductor-linux-arm64"
      sha256 "4a04ac8cea06129c0672fbbe390547dc4ad762fcc868970498ba36ec65028f30"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
