using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Windows;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x0200000D RID: 13
	public class ExportTools
	{
		// Token: 0x06000065 RID: 101 RVA: 0x00005C90 File Offset: 0x00003E90
		public static bool ExportXElementToCSV(List<XElement> XElementList, string FilePath)
		{
			bool flag = true;
			StreamWriter streamWriter = null;
			try
			{
				streamWriter = new StreamWriter(FilePath, false, Encoding.UTF8);
				foreach (XAttribute xattribute in XElementList[0].Attributes())
				{
					string text = string.Empty;
					string text2;
					if ((text2 = xattribute.Name.LocalName.ToUpper()) == null)
					{
						goto IL_04CD;
					}
					if (<PrivateImplementationDetails>{651016A1-3DF1-4842-8AD5-994C2F6A340C}.$$method0x600005f-1 == null)
					{
						<PrivateImplementationDetails>{651016A1-3DF1-4842-8AD5-994C2F6A340C}.$$method0x600005f-1 = new Dictionary<string, int>(26)
						{
							{ "COLUMN", 0 },
							{ "NAME", 1 },
							{ "VER", 2 },
							{ "TABLE", 3 },
							{ "ATTRIBUTE", 4 },
							{ "TYPE", 5 },
							{ "REQ", 6 },
							{ "I_ZOOM", 7 },
							{ "C_ZOOM", 8 },
							{ "CHK_REF", 9 },
							{ "PROG_REL", 10 },
							{ "ITEMS", 11 },
							{ "DEFAULT", 12 },
							{ "MAX", 13 },
							{ "MIN", 14 },
							{ "PERCENTAGE", 15 },
							{ "FORMAT", 16 },
							{ "WIDGET", 17 },
							{ "REF_KEY", 18 },
							{ "REF_TABLE", 19 },
							{ "REF_FK", 20 },
							{ "REF_RTN", 21 },
							{ "REF_USAGE", 22 },
							{ "CHK_TABLE", 23 },
							{ "CHK_COL", 24 },
							{ "STATUS", 25 }
						};
					}
					int num;
					if (!<PrivateImplementationDetails>{651016A1-3DF1-4842-8AD5-994C2F6A340C}.$$method0x600005f-1.TryGetValue(text2, out num))
					{
						goto IL_04CD;
					}
					switch (num)
					{
					case 0:
						text = Application.Current.FindResource("specProperty_column") as string;
						break;
					case 1:
						text = Application.Current.FindResource("specProperty_column_name") as string;
						break;
					case 2:
						text = Application.Current.FindResource("specProperty_ver") as string;
						break;
					case 3:
						text = Application.Current.FindResource("specProperty_table") as string;
						break;
					case 4:
						text = Application.Current.FindResource("specProperty_attribute") as string;
						break;
					case 5:
						text = Application.Current.FindResource("specProperty_type") as string;
						break;
					case 6:
						text = Application.Current.FindResource("specProperty_req") as string;
						break;
					case 7:
						text = Application.Current.FindResource("specProperty_i_zoom") as string;
						break;
					case 8:
						text = Application.Current.FindResource("specProperty_c_zoom") as string;
						break;
					case 9:
						text = Application.Current.FindResource("specProperty_chk_ref") as string;
						break;
					case 10:
						text = Application.Current.FindResource("specProperty_prog_rel") as string;
						break;
					case 11:
						text = Application.Current.FindResource("specProperty_items") as string;
						break;
					case 12:
						text = Application.Current.FindResource("specProperty_default") as string;
						break;
					case 13:
						text = Application.Current.FindResource("specProperty_max") as string;
						break;
					case 14:
						text = Application.Current.FindResource("specProperty_min") as string;
						break;
					case 15:
						text = Application.Current.FindResource("specProperty_percentage") as string;
						break;
					case 16:
						text = Application.Current.FindResource("specProperty_format") as string;
						break;
					case 17:
						text = Application.Current.FindResource("specProperty_widget") as string;
						break;
					case 18:
						text = Application.Current.FindResource("specProperty_ref_key") as string;
						break;
					case 19:
						text = Application.Current.FindResource("specProperty_ref_table") as string;
						break;
					case 20:
						text = Application.Current.FindResource("specProperty_ref_fk") as string;
						break;
					case 21:
						text = Application.Current.FindResource("specProperty_ref_rtn") as string;
						break;
					case 22:
						text = Application.Current.FindResource("specProperty_ref_usage") as string;
						break;
					case 23:
						text = Application.Current.FindResource("specProperty_chk_exists_table") as string;
						break;
					case 24:
						text = Application.Current.FindResource("specProperty_chk_exists_column") as string;
						break;
					case 25:
						text = Application.Current.FindResource("specProperty_action_status") as string;
						break;
					default:
						goto IL_04CD;
					}
					IL_04D9:
					streamWriter.Write(text);
					if (XElementList[0].LastAttribute != xattribute)
					{
						streamWriter.Write(",");
						continue;
					}
					continue;
					IL_04CD:
					text = xattribute.Name.LocalName;
					goto IL_04D9;
				}
				streamWriter.Write(string.Format(",{0}", Application.Current.FindResource("menu_Specification")));
				streamWriter.Write(streamWriter.NewLine);
				int count = XElementList.Count;
				for (int i = 0; i < count; i++)
				{
					foreach (XAttribute xattribute2 in XElementList[i].Attributes())
					{
						streamWriter.Write(xattribute2.Value);
						if (XElementList[i].LastAttribute != xattribute2)
						{
							streamWriter.Write(",");
						}
					}
					if (!string.IsNullOrEmpty(XElementList[i].Value))
					{
						streamWriter.Write(",");
						streamWriter.Write(string.Format("\"{0}\"", XElementList[i].Value));
					}
					streamWriter.Write(streamWriter.NewLine);
				}
			}
			catch (IOException)
			{
				DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_ExportSpecError") as string, FilePath));
				flag = false;
			}
			catch
			{
				DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_ExportSaveSpecError") as string, FilePath));
				flag = false;
			}
			finally
			{
				if (streamWriter != null)
				{
					streamWriter.Close();
				}
			}
			return flag;
		}
	}
}
