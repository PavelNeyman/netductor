class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.102"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.102/netductor-op-darwin-arm64"
      sha256 "1b83f4b839ecb19526dbea9c44673f157246f4d4068e4ebcc3116610b7a5c870"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.102/netductor-op-linux-amd64"
      sha256 "b80e99792c42191cdd98dc76c0bf679b7480c117025c404f7791cde1e97191d7"
    end
  end

  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end

  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
