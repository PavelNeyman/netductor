class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.343"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.343/netductor-op-darwin-arm64"
      sha256 "94e87744d4fe88096e2ace7bef39fde63db8c86b6ac360d70c8f6cddc7a3bb6a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.343/netductor-op-darwin-amd64"
      sha256 "66522a6859e503167011f29872ecacb4a216c015b9f5f1a079aa43602ee67251"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.343/netductor-op-linux-amd64"
      sha256 "3f4433fcddcd91af86303fd803768d49b4275b83a710bab7330e222d4304d01f"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
