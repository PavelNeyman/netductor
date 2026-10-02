class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.173"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.173/netductor-op-darwin-arm64"
      sha256 "67c72d29616fe009ffe29164575e621f115140c64d8b2776f1fcb73762ed5464"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.173/netductor-op-darwin-amd64"
      sha256 "cb3381b19c36456e95b6c907840003b06487a0125580bb5d1b41d7ee0f18d5cf"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.173/netductor-op-linux-amd64"
      sha256 "75f8a2594da6560ac4d5c9ca91723a0d69aba4ad4c3959610dfaad8b6dc73f25"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
