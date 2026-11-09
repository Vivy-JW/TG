using System;
using System.Globalization;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesigner
{
	// Token: 0x02000006 RID: 6
	public class NumbericValidationRule : ValidationRule
	{
		// Token: 0x06000041 RID: 65 RVA: 0x00002D5C File Offset: 0x00000F5C
		public override ValidationResult Validate(object value, CultureInfo cultureInfo)
		{
			string text = value as string;
			if (text == null)
			{
				return new ValidationResult(false, "Please enter some text");
			}
			if (Regex.IsMatch(text, this.CheckReg))
			{
				return new ValidationResult(false, Application.Current.FindResource("Message_OnlyAllowNumber") as string);
			}
			return new ValidationResult(true, null);
		}

		// Token: 0x04000022 RID: 34
		private readonly string CheckReg = "[^0-9\\\\s]";
	}
}
