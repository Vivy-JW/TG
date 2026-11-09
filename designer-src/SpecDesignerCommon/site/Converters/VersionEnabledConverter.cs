using System;
using System.Globalization;
using System.Reflection;
using System.Windows.Data;

namespace SpecDesignerCommon.Site.Converters
{
	// Token: 0x02000084 RID: 132
	public class VersionEnabledConverter : IValueConverter
	{
		// Token: 0x0600053B RID: 1339 RVA: 0x000181C0 File Offset: 0x000163C0
		public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
		{
			if (this._entryVersion == null)
			{
				this._entryVersion = Assembly.GetEntryAssembly().GetName().Version;
			}
			string text = (string)value;
			if (string.IsNullOrEmpty(text))
			{
				return false;
			}
			string text2 = string.Empty;
			try
			{
				text2 = RegistryReader.GetExeDir(text);
			}
			catch (MissingFieldException)
			{
				return false;
			}
			return !string.IsNullOrEmpty(text2);
		}

		// Token: 0x0600053C RID: 1340 RVA: 0x00018240 File Offset: 0x00016440
		public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}

		// Token: 0x040001F7 RID: 503
		private Version _entryVersion;
	}
}
