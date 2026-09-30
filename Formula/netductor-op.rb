class Netductor < Formula
  desc "Netductor operator (Mac/Linux CLI+TUI+Web)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.128"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.128/netductor-op-darwin-arm64"
      sha256 "f0e55aa52690bf4953f6eaca9d6d59bb73a6f59871d1aa5a2741f21d646000ea"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.128/netductor-op-linux-amd64"
      sha256 "9b0dc972cbc7f444989f07c7d7ffceea414ad99e390935441744a163346ff8fd"
    end
  end

  def install
    bin.install Dir["netductor-op*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end

  test do
    assert_match "0.9.128", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
