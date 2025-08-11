/*
 * Copyright (C) The Arvados Authors. All rights reserved.
 *
 * SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0
 *
 */

package org.arvados.client.api.model.argument;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonPropertyOrder;

@JsonFormat(shape = JsonFormat.Shape.ARRAY)
@JsonInclude(JsonInclude.Include.NON_NULL)
@JsonPropertyOrder({ "attribute", "operator", "operand" })
public class Filter {

    @JsonProperty("attribute")
    private String attribute;

    @JsonProperty("operator")
    private Operator operator;

    @JsonProperty("operand")
    private Object operand;

    private Filter(String attribute, Operator operator, Object operand) {
        this.attribute = attribute;
        this.operator = operator;
        this.operand = operand;
    }

    public static Filter of(String attribute, Operator operator, Object operand) {
        return new Filter(attribute, operator, operand);
    }

    public String getAttribute() {
        return this.attribute;
    }

    public Operator getOperator() {
        return this.operator;
    }

    public Object getOperand() {
        return this.operand;
    }

    public boolean equals(Object o) {
        if (o == this) return true;
        if (!(o instanceof Filter)) return false;
        final Filter other = (Filter) o;
        final Object this$attribute = this.getAttribute();
        final Object other$attribute = other.getAttribute();
        if (this$attribute == null ? other$attribute != null : !this$attribute.equals(other$attribute)) return false;
        final Object this$operator = this.getOperator();
        final Object other$operator = other.getOperator();
        if (this$operator == null ? other$operator != null : !this$operator.equals(other$operator)) return false;
        final Object this$operand = this.getOperand();
        final Object other$operand = other.getOperand();
        if (this$operand == null ? other$operand != null : !this$operand.equals(other$operand)) return false;
        return true;
    }

    public int hashCode() {
        final int PRIME = 59;
        int result = 1;
        final Object $attribute = this.getAttribute();
        result = result * PRIME + ($attribute == null ? 43 : $attribute.hashCode());
        final Object $operator = this.getOperator();
        result = result * PRIME + ($operator == null ? 43 : $operator.hashCode());
        final Object $operand = this.getOperand();
        result = result * PRIME + ($operand == null ? 43 : $operand.hashCode());
        return result;
    }

    public String toString() {
        return "Filter(attribute=" + this.getAttribute() + ", operator=" + this.getOperator() + ", operand=" + this.getOperand() + ")";
    }

    public enum Operator {

        @JsonProperty("<")
        LESS,

        @JsonProperty("<=")
        LESS_EQUALS,

        @JsonProperty(">=")
        MORE_EQUALS,

        @JsonProperty(">")
        MORE,

        @JsonProperty("like")
        LIKE,

        @JsonProperty("ilike")
        ILIKE,

        @JsonProperty("=")
        EQUALS,

        @JsonProperty("!=")
        NOT_EQUALS,

        @JsonProperty("in")
        IN,

        @JsonProperty("not in")
        NOT_IN,

        @JsonProperty("is_a")
        IS_A,

        @JsonProperty("exists")
        EXISTS
    }
}
