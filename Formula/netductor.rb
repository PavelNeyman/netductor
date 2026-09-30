class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.124"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.124/netductor-op-darwin-arm64"
      sha256 "08d4c112978dd49a468ced08aea67768558bd4f869849e5c9b35bc26ac8de1aa"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.124/netductor-op-darwin-amd64"
      sha256 "f7f9870b43c469a5eea593cf7a831eefa282ab3dd3154f598642bb20d4c165d4"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.124/netductor-op-linux-amd64"
      sha256 "41e9b52f913c6338591ec9edf6c0f1e5503b1a439c7984969213d96b9da8aa52"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
