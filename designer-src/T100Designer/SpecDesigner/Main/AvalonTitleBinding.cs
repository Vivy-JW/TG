using System;
using System.Windows;

namespace SpecDesigner.Main
{
	// Token: 0x0200001A RID: 26
	public class AvalonTitleBinding : DependencyObject
	{
		// Token: 0x06000219 RID: 537 RVA: 0x0000A61D File Offset: 0x0000881D
		public static string GetTitle(DependencyObject obj)
		{
			return (string)obj.GetValue(AvalonTitleBinding.TitleProperty);
		}

		// Token: 0x0600021A RID: 538 RVA: 0x0000A62F File Offset: 0x0000882F
		public static void SetTitle(DependencyObject obj, string value)
		{
			obj.SetValue(AvalonTitleBinding.TitleProperty, value);
		}

		// Token: 0x040000E9 RID: 233
		public static readonly DependencyProperty TitleProperty = DependencyProperty.RegisterAttached("Title", typeof(string), typeof(AvalonTitleBinding), new UIPropertyMetadata(string.Empty));
	}
}
