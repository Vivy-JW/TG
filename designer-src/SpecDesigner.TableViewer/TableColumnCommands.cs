using System;
using System.Linq;
using System.Text;
using System.Windows;
using System.Windows.Input;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.TableViewer
{
	// Token: 0x02000004 RID: 4
	public class TableColumnCommands
	{
		// Token: 0x06000019 RID: 25 RVA: 0x00002A28 File Offset: 0x00000C28
		public static void CanExecuteCopyContent(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			string text = e.Parameter as string;
			e.CanExecute = text.Length > 0;
		}

		// Token: 0x0600001A RID: 26 RVA: 0x00002A58 File Offset: 0x00000C58
		public static void ExecutedCopyContent(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			string text = e.Parameter as string;
			Clipboard.Clear();
			Clipboard.SetDataObject(text);
		}

		// Token: 0x0600001B RID: 27 RVA: 0x00002A84 File Offset: 0x00000C84
		public static void CanExecuteCopyAllColumnsName(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			string text = e.Parameter as string;
			if (text.Length > 0 && TableColumnHelper.GetColFields(text).Count<XElement>() > 0)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x0600001C RID: 28 RVA: 0x00002ACC File Offset: 0x00000CCC
		public static void ExecutedCopyAllColumnsName(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			string text = e.Parameter as string;
			StringBuilder stringBuilder = new StringBuilder();
			foreach (XElement xelement in TableColumnHelper.GetColFields(text))
			{
				XAttribute xattribute = xelement.Attribute("name");
				if (xattribute != null)
				{
					if (stringBuilder.Length > 0)
					{
						stringBuilder.Append(",");
					}
					stringBuilder.Append(xelement.Attribute("name").Value);
				}
			}
			if (stringBuilder.Length > 0)
			{
				Clipboard.Clear();
				Clipboard.SetDataObject(stringBuilder.ToString());
			}
		}

		// Token: 0x0600001D RID: 29 RVA: 0x00002B90 File Offset: 0x00000D90
		public static void CanExecuteCopyAllColumnsToRecord(object sender, CanExecuteRoutedEventArgs e)
		{
			e.Handled = true;
			string text = e.Parameter as string;
			if (text.Length > 0 && TableColumnHelper.GetColFields(text).Count<XElement>() > 0)
			{
				e.CanExecute = true;
				return;
			}
			e.CanExecute = false;
		}

		// Token: 0x0600001E RID: 30 RVA: 0x00002BD8 File Offset: 0x00000DD8
		public static void ExecutedCopyAllColumnsToRecord(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			string text = e.Parameter as string;
			StringBuilder stringBuilder = new StringBuilder();
			stringBuilder.Append(string.Concat(new string[]
			{
				"DEFINE l_",
				text.Split(new char[] { '_' })[0],
				" RECORD  #",
				TableColumnHelper.GetTableDesc(text),
				"\r\n"
			}));
			foreach (XElement xelement in TableColumnHelper.GetColFields(text))
			{
				XAttribute xattribute = xelement.Attribute("name");
				if (xattribute != null)
				{
					string columnText = TableColumnHelper.GetColumnText(text, xattribute.Value);
					stringBuilder.Append(string.Concat(new string[]
					{
						"       ",
						xelement.Attribute("name").Value,
						" LIKE ",
						text,
						".",
						xattribute.Value,
						", #",
						columnText,
						"\r\n"
					}));
				}
			}
			stringBuilder.Append("END RECORD");
			if (stringBuilder.Length > 0)
			{
				Clipboard.Clear();
				Clipboard.SetDataObject(stringBuilder.ToString());
			}
		}

		// Token: 0x0400000F RID: 15
		public static RoutedCommand CopyContentCommand = new RoutedCommand();

		// Token: 0x04000010 RID: 16
		public static RoutedCommand CopyAllColumnsNameCommand = new RoutedCommand();

		// Token: 0x04000011 RID: 17
		public static RoutedCommand CopyAllColumnsToRecordCommand = new RoutedCommand();
	}
}
