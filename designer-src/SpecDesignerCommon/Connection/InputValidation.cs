using System;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x0200013C RID: 316
	public class InputValidation : ValidationRule
	{
		// Token: 0x06000B09 RID: 2825 RVA: 0x00035718 File Offset: 0x00033918
		public override ValidationResult Validate(object value, CultureInfo cultureInfo)
		{
			if (value == null)
			{
				return new ValidationResult(false, Application.Current.FindResource("Message_CantNull") as string);
			}
			if (string.IsNullOrEmpty(value.ToString()))
			{
				return new ValidationResult(false, Application.Current.FindResource("Message_CantEmpty") as string);
			}
			string text = value.ToString();
			if (text.Contains("."))
			{
				return new ValidationResult(false, Application.Current.FindResource("Message_CantContainDot") as string);
			}
			return new ValidationResult(true, null);
		}
	}
}
