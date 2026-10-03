class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.200"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.200/netductor-op-darwin-arm64"
      sha256 "2bbe917c0637a14dba2389eb66a38172fb3669a1bd63527cd401ab84e0c2da4a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.200/netductor-op-darwin-amd64"
      sha256 "2aae2f73ef313946b9b8d822c139b0875747f26d3af4e6209e111c1b359eb979"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.200/netductor-op-linux-amd64"
      sha256 "c5bca48dab545ae7462cfe174da1b9b712ffe69cca0eae9fbc8cbeddd57c1e57"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
