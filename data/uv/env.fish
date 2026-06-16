if not contains "data/uv" $PATH
    # Prepending path in case a system-installed binary needs to be overridden
    set -x PATH "data/uv" $PATH
end
