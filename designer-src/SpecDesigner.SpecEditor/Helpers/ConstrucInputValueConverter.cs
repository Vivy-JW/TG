using System;
using System.Globalization;
using System.Windows.Data;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000026 RID: 38
	public class ConstrucInputValueConverter : IValueConverter
	{
		// Token: 0x06000113 RID: 275 RVA: 0x000097AC File Offset: 0x000079AC
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			this.node = value as SpecFieldNode;
			if (this.node == null)
			{
				return null;
			}
			if (this.node.CanQuery.Equals("Y", StringComparison.CurrentCultureIgnoreCase) && this.node.CanEdit.Equals("N", StringComparison.CurrentCultureIgnoreCase))
			{
				return "CONSTRUCT";
			}
			if (this.node.CanQuery.Equals("N", StringComparison.CurrentCultureIgnoreCase) && this.node.CanEdit.Equals("Y", StringComparison.CurrentCultureIgnoreCase))
			{
				return "INPUT";
			}
			if (this.node.CanQuery.Equals("N", StringComparison.CurrentCultureIgnoreCase) && this.node.CanEdit.Equals("N", StringComparison.CurrentCultureIgnoreCase))
			{
				return "NONE";
			}
			return null;
		}

		// Token: 0x06000114 RID: 276 RVA: 0x00009874 File Offset: 0x00007A74
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			string text = value as string;
			if (text.Equals("INPUT", StringComparison.CurrentCultureIgnoreCase))
			{
				this.node.CanQuery = "N";
				this.node.CanEdit = "Y";
			}
			else if (text.Equals("CONSTRUCT", StringComparison.CurrentCultureIgnoreCase))
			{
				this.node.CanQuery = "Y";
				this.node.CanEdit = "N";
			}
			else
			{
				this.node.CanQuery = "N";
				this.node.CanEdit = "N";
			}
			return this.node;
		}

		// Token: 0x04000092 RID: 146
		private SpecFieldNode node;
	}
}
