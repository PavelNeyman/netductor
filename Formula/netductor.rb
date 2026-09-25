class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.9"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.9/netductor-op-darwin-arm64"
      sha256 "ee094e1106abd5ee1f921a0e1600fe63346506a8151f5e6cce8ec7aaa5bae3df"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.9/netductor-op-darwin-amd64"
      sha256 "48ff912d045fc9ab94482cc56466d3c13758df1ff0012a21d48173ed0a08647e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.9/netductor-op-linux-amd64"
      sha256 "75fd70274bc202b306278f6b8f79ac0e9cf1f2ca7653db121623db1081e74263"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.9/netductor-op-linux-arm64"
      sha256 "8e0d7552b17c04d6c476e5d65ea87674b6c8f09a16152354f59a758cce973b73"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
