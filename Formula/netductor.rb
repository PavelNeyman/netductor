class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.178"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.178/netductor-op-darwin-arm64"
      sha256 "8c773d8eccb69ed8ad0ca4be14348737b14e2d390cbc9495ee3fb7b317c6414f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.178/netductor-op-darwin-amd64"
      sha256 "665ee8b5aac8d87ad64db4a7be7fea6b5822e0ecb88c0c0113a540ff5eaa7516"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.178/netductor-op-linux-amd64"
      sha256 "06d98df524415df23bae161ad6ddd4f59a1cbb2768637b49c994601dcf263b7a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
