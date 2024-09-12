#! /usr/bin/env bats
#
# Copyright (c) 2024, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
# bats file_tags=APPLICATION

@test "Generating a template for an application emits reasonable information" {
	run ocne application template --name grafana
	[ $status -eq 0 ]
	echo "$output" | grep 'repository:'
}

@test "Generating a template for an application emits reasonable information from embedded catalog" {
	run ocne application template --name grafana --catalog embedded
	[ $status -eq 0 ]
	echo "$output" | grep 'repository:'
}

@test "Generating a template in interactive mode emits reasonable information" {
    SUFFIX=$RANDOM
    TEMPLATE_OUTPUT_SCRIPT=$HOME/template_output-$SUFFIX.sh
    TEMPLATE_OUTPUT=$HOME/template_output_$SUFFIX

	cat > $TEMPLATE_OUTPUT_SCRIPT << EOF
#! /usr/bin/env bash
cat \$1 > $TEMPLATE_OUTPUT
EOF
	chmod +x $TEMPLATE_OUTPUT_SCRIPT
	export EDITOR=$TEMPLATE_OUTPUT_SCRIPT

	run ocne application template --interactive --name grafana
	[ $status -eq 0 ]
	echo "$output" | grep -v 'repository:'
	grep 'repository' $TEMPLATE_OUTPUT

	rm $TEMPLATE_OUTPUT_SCRIPT
}
