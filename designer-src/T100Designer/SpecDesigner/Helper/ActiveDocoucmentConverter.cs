using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesigner.ViewModels;

namespace SpecDesigner.Helper
{
	// Token: 0x0200002E RID: 46
	public class ActiveDocoucmentConverter : IValueConverter
	{
		// Token: 0x06000279 RID: 633 RVA: 0x0000BA84 File Offset: 0x00009C84
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (EditorWorkspace.This.ActiveDocument != null)
			{
				return EditorWorkspace.This.ActiveDocument.UI;
			}
			return null;
		}

		// Token: 0x0600027A RID: 634 RVA: 0x0000BAA3 File Offset: 0x00009CA3
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
