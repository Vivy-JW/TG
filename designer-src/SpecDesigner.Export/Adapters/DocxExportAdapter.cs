using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Drawing;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;
using System.Threading;
using System.Windows;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using System.Windows.Resources;
using System.Windows.Threading;
using System.Xml.Linq;
using Microsoft.Win32;
using Novacode;
using SpecDesigner.Controls.AdornedControl;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Export.Properties;
using SpecDesigner.FormEditor;
using SpecDesigner.FormEditor.Helpers;
using SpecDesigner.FormEditor.Views;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.Export.Adapters
{
	// Token: 0x02000004 RID: 4
	public class DocxExportAdapter : IExportAdapter
	{
		// Token: 0x06000026 RID: 38 RVA: 0x00002368 File Offset: 0x00000568
		public DocxExportAdapter(FormEditorMainWindow form)
		{
			this._mf = VisualTreeHelperEx.FindChild<ManagedForm>(form);
			this._programKey = form.ProgramKey;
		}

		// Token: 0x06000027 RID: 39 RVA: 0x000023B8 File Offset: 0x000005B8
		public void Export(string templateFolder)
		{
			if (this._mf == null)
			{
				DesignerMessageBox.Show(Strings.DocxExport_FormNotFound);
				return;
			}
			SaveFileDialog saveFileDialog = new SaveFileDialog();
			saveFileDialog.FileName = this._programKey.Program;
			saveFileDialog.DefaultExt = ".docx";
			saveFileDialog.Filter = "Word document (.docx)|*.docx";
			if (saveFileDialog.ShowDialog() == true)
			{
				string text = Path.Combine(templateFolder, "template.docx");
				StreamResourceInfo resourceStream = Application.GetResourceStream(new Uri("pack://application:,,,/SpecDesigner.Export;component/template.docx"));
				Stream stream = resourceStream.Stream;
				if (!File.Exists(text))
				{
					BinaryReader binaryReader = new BinaryReader(stream);
					FileStream fileStream = new FileStream(text, FileMode.Create);
					BinaryWriter binaryWriter = new BinaryWriter(fileStream);
					byte[] array = new byte[stream.Length];
					stream.Read(array, 0, array.Length);
					binaryWriter.Write(array);
					binaryReader.Close();
					binaryWriter.Close();
					stream.Close();
				}
				File.Copy(text, saveFileDialog.FileName, true);
				using (DocX docX = DocX.Load(saveFileDialog.FileName))
				{
					this.RenderDocumentInfo(docX);
					this.RenderProgramInfo(docX);
					this.RenderSnapshot(docX);
					this.RenderFields(docX);
					this.RenderAction(docX);
					docX.Save();
				}
				try
				{
					Process.Start(new ProcessStartInfo
					{
						FileName = "WINWORD.EXE",
						Arguments = saveFileDialog.FileName
					});
				}
				catch (Exception)
				{
					Process.Start(templateFolder);
				}
			}
		}

		// Token: 0x06000028 RID: 40 RVA: 0x00002554 File Offset: 0x00000754
		private void RenderAction(DocX document)
		{
			Paragraph paragraph = document.InsertParagraph(Strings.DocxExport_ActionSpecDescription).Bold();
			paragraph.InsertPageBreakBeforeSelf();
			Table table = document.AddTable(1, 2, false);
			table.SetFirstColumn(false);
			table.Design = TableDesign.LightGridAccent1;
			table.AutoFit = AutoFit.Window;
			paragraph.InsertTableAfterSelf(table);
			table.Rows[0].Cells[0].Paragraphs[0].Append(Strings.DocxExport_ActionName).Bold();
			table.Rows[0].Cells[1].Paragraphs[0].Append(Strings.DocxExport_SpecDescription).Bold();
			this.SetFontSize(table.Rows[0], this.fontSize, true);
			foreach (SpecActionNode specActionNode in SettingManager.Get().GetTzpManger(this._programKey).SpecificationInfo.Actions)
			{
				Row row = table.InsertRow();
				row.Cells[0].Paragraphs[0].Append(string.Format("{0}\r\n{1}", specActionNode.Name, specActionNode.LocalString));
				row.Cells[1].Paragraphs[0].Append(specActionNode.CDATA);
				this.SetFontSize(row, this.fontSize, false);
			}
		}

		// Token: 0x06000029 RID: 41 RVA: 0x000026F0 File Offset: 0x000008F0
		private void RenderFields(DocX document)
		{
			Paragraph paragraph = document.InsertParagraph(Strings.DocxExport_FieldSpecDescription).Bold();
			paragraph.InsertPageBreakBeforeSelf();
			Table table = document.AddTable(2, 9, false);
			table.Design = TableDesign.LightGridAccent1;
			table.AutoFit = AutoFit.Window;
			paragraph.InsertTableAfterSelf(table);
			table.Rows[0].Cells[0].Paragraphs[0].Append(Strings.DocxExport_FieldName).Bold();
			table.Rows[0].Cells[1].Paragraphs[0].Append(Strings.DocxExport_Required).Bold();
			table.Rows[0].Cells[2].Paragraphs[0].Append(Strings.DocxExport_Items).Bold();
			table.Rows[0].Cells[3].Paragraphs[0].Append(Strings.DocxExport_DefaultValue).Bold();
			table.Rows[0].Cells[4].Paragraphs[0].Append(Strings.DocxExport_MaxValue).Bold();
			table.Rows[0].Cells[5].Paragraphs[0].Append(Strings.DocxExport_MinValue).Bold();
			table.Rows[0].Cells[6].Paragraphs[0].Append(Strings.DocxExport_IZoom).Bold();
			table.Rows[0].Cells[7].Paragraphs[0].Append(Strings.DocxExport_CZoom).Bold();
			table.Rows[0].Cells[8].Paragraphs[0].Append(Strings.DocxExport_ChkRef).Bold();
			table.MergeCellsInColumn(0, 0, 1);
			table.Rows[1].MergeCells(1, 8);
			table.Rows[1].Cells[1].Paragraphs.ToList<Paragraph>().ForEach(delegate(Paragraph pa)
			{
				pa.Remove(false);
			});
			table.Rows[1].Cells[1].Paragraphs[0].Append(Strings.DocxExport_SpecDescription).Bold();
			this.SetFontSize(table.Rows[0], this.fontSize, true);
			this.SetFontSize(table.Rows[1], this.fontSize, true);
			XNamespace xnamespace = XNamespace.Get("http://schemas.openxmlformats.org/wordprocessingml/2006/main");
			table.Rows[0].Xml.Add(new XElement(xnamespace + "trPr", new XElement(xnamespace + "tblHeader")));
			table.Rows[1].Xml.Add(new XElement(xnamespace + "trPr", new XElement(xnamespace + "tblHeader")));
			int num = 2;
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair in SettingManager.Get().GetTzpManger(this._programKey).SpecificationInfo.FormSpeDictionary)
			{
				SpecFieldNode specField = keyValuePair.Value.SpecField;
				if (specField != null)
				{
					string text = string.Format("{0}\r\n{1}", specField.Name, TableColumnHelper.GetColumnTextByFullName(specField.Name));
					string req = specField.Req;
					string widget = specField.Widget;
					string items = specField.Items;
					string @default = specField.Default;
					string max = specField.Max;
					string min = specField.Min;
					string izoom = specField.IZoom;
					string czoom = specField.CZoom;
					string chkRef = specField.ChkRef;
					string cdata = specField.CDATA;
					table.InsertRow();
					table.InsertRow();
					table.Rows[num].Cells[0].Paragraphs[0].Append(text);
					table.Rows[num].Cells[1].Paragraphs[0].Append(req);
					table.Rows[num].Cells[2].Paragraphs[0].Append(items);
					table.Rows[num].Cells[3].Paragraphs[0].Append(@default);
					table.Rows[num].Cells[4].Paragraphs[0].Append(max);
					table.Rows[num].Cells[5].Paragraphs[0].Append(min);
					table.Rows[num].Cells[6].Paragraphs[0].Append(izoom);
					table.Rows[num].Cells[7].Paragraphs[0].Append(czoom);
					table.Rows[num].Cells[8].Paragraphs[0].Append(chkRef);
					table.MergeCellsInColumn(0, num, num + 1);
					table.Rows[num + 1].MergeCells(1, 8);
					table.Rows[num + 1].Cells[1].Paragraphs.ToList<Paragraph>().ForEach(delegate(Paragraph pa)
					{
						pa.Remove(false);
					});
					table.Rows[num + 1].Cells[1].Paragraphs[0].Append(cdata);
					this.SetFontSize(table.Rows[num], this.fontSize, false);
					this.SetFontSize(table.Rows[num + 1], this.fontSize, false);
					num += 2;
				}
			}
		}

		// Token: 0x0600002A RID: 42 RVA: 0x00002D7C File Offset: 0x00000F7C
		private void RenderDocumentInfo(DocX document)
		{
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this._programKey).SpecificationInfo;
			foreach (Header header in new List<Header>
			{
				document.Headers.first,
				document.Headers.odd,
				document.Headers.even
			})
			{
				if (header != null)
				{
					header.ReplaceText("{#TITLE#}", Strings.DocxExport_Header_Title, false, RegexOptions.None, null, null, MatchFormattingOptions.SubsetMatch);
					header.ReplaceText("{#VERSION#}", specificationInfo.Ver, false, RegexOptions.None, null, null, MatchFormattingOptions.SubsetMatch);
					header.ReplaceText("{#DATABASE#}", "Oracle", false, RegexOptions.None, null, null, MatchFormattingOptions.SubsetMatch);
					header.ReplaceText("{#DEPT#}", Strings.DocxExport_Header_Dept, false, RegexOptions.None, null, null, MatchFormattingOptions.SubsetMatch);
					header.ReplaceText("{#SD#}", Strings.DocxExport_Header_SD, false, RegexOptions.None, null, null, MatchFormattingOptions.SubsetMatch);
				}
			}
			foreach (Footer footer in new List<Footer>
			{
				document.Footers.first,
				document.Footers.odd,
				document.Footers.even
			})
			{
				if (footer != null)
				{
					footer.ReplaceText("{#COMMENT#}", Strings.DocxExport_Footer_Comment, false, RegexOptions.None, null, null, MatchFormattingOptions.SubsetMatch);
					footer.ReplaceText("{#SA#}", Strings.DocxExport_Header_SA, false, RegexOptions.None, null, null, MatchFormattingOptions.SubsetMatch);
					footer.ReplaceText("{#SD#}", Strings.DocxExport_Header_SD, false, RegexOptions.None, null, null, MatchFormattingOptions.SubsetMatch);
				}
			}
		}

		// Token: 0x0600002B RID: 43 RVA: 0x00002F38 File Offset: 0x00001138
		private void RenderSnapshot(DocX document)
		{
			document.InsertParagraph(Strings.DocxExport_ProgramCapture).Bold().InsertPageBreakBeforeSelf();
			this.capture(document, this._mf.MainForm);
			List<AdornedControl> list = new List<AdornedControl>();
			VisualTreeHelperEx.FindAllChild<AdornedControl>(this._mf.MainForm, list);
			foreach (AdornedControl adornedControl in list)
			{
				XmlElement xmlElement = adornedControl.DataContext as XmlElement;
				if (xmlElement != null && xmlElement.Type == ComponentType.Folder)
				{
					foreach (XmlElement xmlElement2 in xmlElement.Nodes)
					{
						xmlElement2.IsSelected = true;
						double actualWidth = adornedControl.ActualWidth;
						double actualHeight = adornedControl.ActualHeight;
						document.InsertParagraph(xmlElement2.LocalString + "：").Bold();
						adornedControl.Measure(new global::System.Windows.Size(actualWidth, actualHeight));
						adornedControl.Arrange(new Rect(0.0, 0.0, actualWidth, actualHeight));
						adornedControl.Dispatcher.Invoke(DispatcherPriority.Render, DocxExportAdapter.EmptyDelegate);
						Thread.Sleep(500);
						this.capture(document, adornedControl);
					}
				}
			}
		}

		// Token: 0x0600002C RID: 44 RVA: 0x000030AC File Offset: 0x000012AC
		private void capture(DocX document, FrameworkElement fe)
		{
			if (fe.RenderSize == new global::System.Windows.Size(0.0, 0.0))
			{
				return;
			}
			RenderTargetBitmap renderTargetBitmap = new RenderTargetBitmap((int)fe.ActualWidth, (int)fe.ActualHeight, 96.0, 96.0, PixelFormats.Pbgra32);
			renderTargetBitmap.Render(fe);
			global::Novacode.Image image;
			using (MemoryStream memoryStream = new MemoryStream())
			{
				new PngBitmapEncoder
				{
					Interlace = PngInterlaceOption.On,
					Frames = { BitmapFrame.Create(renderTargetBitmap) }
				}.Save(memoryStream);
				image = document.AddImage(memoryStream);
			}
			Paragraph paragraph = document.InsertParagraph();
			Picture picture = image.CreatePicture();
			float num = document.PageWidth - (document.MarginLeft + document.MarginRight);
			if ((float)picture.Width > num)
			{
				int width = picture.Width;
				float num2 = (float)picture.Height;
				float num3 = num / (float)picture.Width;
				int num4 = (int)num;
				int num5 = (int)(num2 * num3);
				picture.Height = num5;
				picture.Width = num4;
			}
			paragraph.InsertPicture(picture, 0);
		}

		// Token: 0x0600002D RID: 45 RVA: 0x000031DC File Offset: 0x000013DC
		private void RenderProgramInfo(DocX document)
		{
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this._programKey).SpecificationInfo;
			Paragraph paragraph = document.Paragraphs.FirstOrDefault<Paragraph>();
			paragraph.Append(Strings.DocxExport_ProgramID).Bold().AppendLine("\t" + this._programKey.Program);
			document.InsertParagraph().Append(Strings.DocxExport_ProgramDescription).Bold()
				.AppendLine(specificationInfo.ProgramSpec.CDATA + " ")
				.AppendLine(" ");
			document.InsertParagraph().Append(Strings.DocxExport_MasterInputSpec).Bold()
				.AppendLine(specificationInfo.ProgramMISpec.CDATA + " ")
				.AppendLine(" ");
			document.InsertParagraph().Append(Strings.DocxExport_DetailInputSpec).Bold()
				.AppendLine(specificationInfo.ProgramDISpec.CDATA + " ")
				.AppendLine(" ");
			document.InsertParagraph().Append(Strings.DocxExport_DetailDisplaySpec).Bold()
				.AppendLine(specificationInfo.ProgramDBSpec.CDATA + " ")
				.AppendLine(" ");
			document.InsertParagraph().AppendLine(Strings.DocxExport_MasterTable).Bold();
		}

		// Token: 0x0600002E RID: 46 RVA: 0x00003398 File Offset: 0x00001598
		private void SetFontSize(Row row, int size, bool bold)
		{
			row.Cells.ForEach(delegate(Cell c)
			{
				if (c.Paragraphs[0].Text.Length > 0)
				{
					c.Paragraphs[0].FontSize((double)size);
				}
				if (bold)
				{
					c.Paragraphs[0].Bold();
				}
			});
		}

		// Token: 0x04000003 RID: 3
		private ManagedForm _mf;

		// Token: 0x04000004 RID: 4
		private PackageKey _programKey;

		// Token: 0x04000005 RID: 5
		private global::System.Drawing.Color bgColor = global::System.Drawing.Color.FromArgb(224, 255, 255);

		// Token: 0x04000006 RID: 6
		private int fontSize = 10;

		// Token: 0x04000007 RID: 7
		private static Action EmptyDelegate = delegate
		{
		};
	}
}
